package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Odds API Types ──

type oddsSport struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type oddsEvent struct {
	ID           string        `json:"id"`
	SportKey     string        `json:"sport_key"`
	SportTitle   string        `json:"sport_title"`
	CommenceTime string        `json:"commence_time"`
	HomeTeam     string        `json:"home_team"`
	AwayTeam     string        `json:"away_team"`
	Bookmakers   []oddsBookmaker `json:"bookmakers"`
}

type oddsBookmaker struct {
	Key        string       `json:"key"`
	Title      string       `json:"title"`
	LastUpdate string       `json:"last_update"`
	Markets    []oddsMarket `json:"markets"`
}

type oddsMarket struct {
	Key        string        `json:"key"`
	LastUpdate string        `json:"last_update"`
	Outcomes   []oddsOutcome `json:"outcomes"`
}

type oddsOutcome struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Point *float64 `json:"point,omitempty"`
}

// ── Sport key → icon mapping ──

var sportGroupToIcon = map[string]string{
	"American Football": "football",
	"Basketball":        "basketball",
	"Baseball":          "baseball",
	"Ice Hockey":        "hockey",
	"Soccer":            "soccer",
	"Tennis":            "tennis",
	"MMA":               "mma",
	"Boxing":            "boxing",
	"Golf":              "golf",
	"Cricket":           "cricket",
	"Rugby League":      "rugby",
	"Rugby Union":       "rugby",
	"Aussie Rules":      "aussie-rules",
}

// ── OddsAPIConnector ──

// OddsAPIConnector syncs sportsbook data from The Odds API.
type OddsAPIConnector struct {
	pool    *pgxpool.Pool
	baseURL string
	apiKey  string
	logger  *slog.Logger
	client  *http.Client
	// Sports to sync (Odds API keys). If empty, syncs top sports.
	sportKeys []string
}

// NewOddsAPIConnector creates a new Odds API connector.
func NewOddsAPIConnector(pool *pgxpool.Pool, apiKey string, logger *slog.Logger) *OddsAPIConnector {
	return &OddsAPIConnector{
		pool:    pool,
		baseURL: "https://api.the-odds-api.com",
		apiKey:  apiKey,
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
		sportKeys: []string{
			"americanfootball_nfl",
			"basketball_nba",
			"baseball_mlb",
			"icehockey_nhl",
			"soccer_usa_mls",
			"soccer_epl",
			"mma_mixed_martial_arts",
			"tennis_atp_french_open",
		},
	}
}

// StartSync begins periodic syncing of sports events and odds.
func (c *OddsAPIConnector) StartSync(ctx context.Context) {
	c.logger.Info("odds api connector starting", "sports", c.sportKeys)

	// Initial sync
	go func() {
		if err := c.syncAll(ctx); err != nil {
			c.logger.Error("odds api initial sync error", "error", err)
		}
	}()

	// Sync events + odds every 10 minutes (conserve free-tier quota)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("odds api sync stopped")
				return
			case <-ticker.C:
				if err := c.syncAll(ctx); err != nil {
					c.logger.Error("odds api sync error", "error", err)
				}
			}
		}
	}()
}

// ── HTTP helper ──

func (c *OddsAPIConnector) oddsGet(ctx context.Context, path string) ([]byte, int, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	if strings.Contains(path, "?") {
		url += "&apiKey=" + c.apiKey
	} else {
		url += "?apiKey=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	remaining := resp.Header.Get("x-requests-remaining")
	c.logger.Debug("odds api request", "path", path, "status", resp.StatusCode, "remaining", remaining)

	if resp.StatusCode == 429 {
		return nil, resp.StatusCode, fmt.Errorf("odds api quota exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("odds api returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	return body, resp.StatusCode, nil
}

// ── Sync All ──

func (c *OddsAPIConnector) syncAll(ctx context.Context) error {
	// Step 1: Sync sports list (free, no quota)
	if err := c.syncSports(ctx); err != nil {
		c.logger.Error("odds api sync sports failed", "error", err)
	}

	// Step 2: For each sport key, fetch events + odds
	totalEvents := 0
	for _, sportKey := range c.sportKeys {
		n, err := c.syncSportEvents(ctx, sportKey)
		if err != nil {
			c.logger.Error("odds api sync sport events failed", "sport", sportKey, "error", err)
			continue
		}
		totalEvents += n
	}

	c.logger.Info("odds api sync complete", "events_synced", totalEvents)
	return nil
}

// ── Sync Sports ──

func (c *OddsAPIConnector) syncSports(ctx context.Context) error {
	body, _, err := c.oddsGet(ctx, "/v4/sports/")
	if err != nil {
		return err
	}

	var sports []oddsSport
	if err := json.Unmarshal(body, &sports); err != nil {
		return fmt.Errorf("decode sports: %w", err)
	}

	for _, s := range sports {
		if !s.Active {
			continue
		}

		icon := sportGroupToIcon[s.Group]
		if icon == "" {
			icon = strings.ToLower(strings.ReplaceAll(s.Group, " ", "-"))
		}

		// Upsert sport by key
		_, err := c.pool.Exec(ctx, `
			INSERT INTO sports (id, key, name, icon, sort_order, active)
			VALUES (gen_random_uuid(), $1, $2, $3, 0, true)
			ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name, active = true`,
			s.Key, s.Title, icon)
		if err != nil {
			c.logger.Warn("odds api upsert sport", "key", s.Key, "error", err)
		}
	}

	return nil
}

// ── Sync Events + Odds for a Sport ──

func (c *OddsAPIConnector) syncSportEvents(ctx context.Context, sportKey string) (int, error) {
	// Fetch odds (costs 1 quota per region per market)
	path := fmt.Sprintf("/v4/sports/%s/odds/?regions=us&markets=h2h,spreads,totals&oddsFormat=decimal&dateFormat=iso", sportKey)
	body, status, err := c.oddsGet(ctx, path)
	if err != nil {
		if status == 429 {
			c.logger.Warn("odds api quota exceeded, stopping sync")
			return 0, nil
		}
		return 0, err
	}

	var events []oddsEvent
	if err := json.Unmarshal(body, &events); err != nil {
		return 0, fmt.Errorf("decode events: %w", err)
	}

	// Get our sport ID
	var sportID uuid.UUID
	err = c.pool.QueryRow(ctx, `SELECT id FROM sports WHERE key = $1`, sportKey).Scan(&sportID)
	if err != nil {
		// Sport might not exist yet — try by creating it
		sportID = uuid.New()
		_, err = c.pool.Exec(ctx, `
			INSERT INTO sports (id, key, name, icon, sort_order, active)
			VALUES ($1, $2, $2, 'sports', 0, true)
			ON CONFLICT (key) DO UPDATE SET active = true RETURNING id`, sportID, sportKey)
		if err != nil {
			return 0, fmt.Errorf("ensure sport %s: %w", sportKey, err)
		}
		c.pool.QueryRow(ctx, `SELECT id FROM sports WHERE key = $1`, sportKey).Scan(&sportID)
	}

	synced := 0
	for _, event := range events {
		if err := c.upsertEvent(ctx, sportID, sportKey, event); err != nil {
			c.logger.Warn("odds api upsert event", "event_id", event.ID, "error", err)
			continue
		}
		synced++
	}

	return synced, nil
}

func (c *OddsAPIConnector) upsertEvent(ctx context.Context, sportID uuid.UUID, sportKey string, event oddsEvent) error {
	// Determine league from sport key
	league := event.SportTitle

	// Parse commence time
	commenceTime, err := time.Parse(time.RFC3339, event.CommenceTime)
	if err != nil {
		return fmt.Errorf("parse commence_time: %w", err)
	}

	// Determine status
	status := "upcoming"
	if time.Now().After(commenceTime) {
		status = "live"
	}

	// Upsert event using odds88_event_id to store the Odds API event ID
	// First try to find existing event by odds88_event_id
	var eventID uuid.UUID
	err = c.pool.QueryRow(ctx, `
		SELECT id FROM sports_events WHERE odds88_event_id = $1`, hashOddsID(event.ID)).Scan(&eventID)
	if err != nil {
		// Create new event
		eventID = uuid.New()
		_, err = c.pool.Exec(ctx, `
			INSERT INTO sports_events (id, sport_id, league, home_team, away_team, start_time, status, odds88_event_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (odds88_event_id) DO UPDATE SET
				home_team = EXCLUDED.home_team,
				away_team = EXCLUDED.away_team,
				start_time = EXCLUDED.start_time,
				status = CASE
					WHEN sports_events.status = 'settled' THEN sports_events.status
					ELSE EXCLUDED.status
				END,
				updated_at = now()`,
			eventID, sportID, league, event.HomeTeam, event.AwayTeam, commenceTime, status, hashOddsID(event.ID))
		if err != nil {
			return fmt.Errorf("upsert event: %w", err)
		}
		// Re-fetch the actual ID in case of conflict
		c.pool.QueryRow(ctx, `SELECT id FROM sports_events WHERE odds88_event_id = $1`, hashOddsID(event.ID)).Scan(&eventID)
	} else {
		// Update existing
		_, _ = c.pool.Exec(ctx, `
			UPDATE sports_events SET
				home_team = $2, away_team = $3, start_time = $4,
				status = CASE WHEN status = 'settled' THEN status ELSE $5 END,
				updated_at = now()
			WHERE id = $1`, eventID, event.HomeTeam, event.AwayTeam, commenceTime, status)
	}

	// Process bookmakers — pick the first one with data (consensus odds)
	if len(event.Bookmakers) == 0 {
		return nil
	}

	// Aggregate markets across bookmakers — use first bookmaker's odds
	bk := event.Bookmakers[0]
	for _, mkt := range bk.Markets {
		if err := c.upsertMarketAndSelections(ctx, eventID, mkt); err != nil {
			c.logger.Debug("odds api upsert market", "event_id", eventID, "market", mkt.Key, "error", err)
		}
	}

	return nil
}

func (c *OddsAPIConnector) upsertMarketAndSelections(ctx context.Context, eventID uuid.UUID, mkt oddsMarket) error {
	// Map Odds API market key to our market type
	marketName := mkt.Key
	marketType := mkt.Key
	switch mkt.Key {
	case "h2h":
		marketName = "Moneyline"
		marketType = "1x2"
	case "spreads":
		marketName = "Spread"
		marketType = "spread"
	case "totals":
		marketName = "Total"
		marketType = "over_under"
	}

	// Generate a deterministic market ID from odds88_market_id
	odds88MarketID := fmt.Sprintf("%s_%s", eventID.String()[:8], mkt.Key)

	// Upsert market
	var marketID uuid.UUID
	err := c.pool.QueryRow(ctx, `
		SELECT id FROM sports_markets WHERE odds88_market_id = $1`, odds88MarketID).Scan(&marketID)
	if err != nil {
		marketID = uuid.New()
		_, err = c.pool.Exec(ctx, `
			INSERT INTO sports_markets (id, event_id, name, type, status, odds88_market_id)
			VALUES ($1, $2, $3, $4, 'open', $5)
			ON CONFLICT (odds88_market_id) DO UPDATE SET
				name = EXCLUDED.name, updated_at = now()`,
			marketID, eventID, marketName, marketType, odds88MarketID)
		if err != nil {
			return fmt.Errorf("upsert market: %w", err)
		}
		c.pool.QueryRow(ctx, `SELECT id FROM sports_markets WHERE odds88_market_id = $1`, odds88MarketID).Scan(&marketID)
	}

	// Upsert selections
	for i, outcome := range mkt.Outcomes {
		selName := outcome.Name
		if outcome.Point != nil {
			if mkt.Key == "spreads" {
				selName = fmt.Sprintf("%s %+.1f", outcome.Name, *outcome.Point)
			} else if mkt.Key == "totals" {
				selName = fmt.Sprintf("%s %.1f", outcome.Name, *outcome.Point)
			}
		}

		// Convert decimal odds to integer (1.75 → 175)
		oddsDecimal := int(outcome.Price * 100)

		// Deterministic selection ID
		odds88SelectionID := int64(hashOddsID(fmt.Sprintf("%s_%s_%d", odds88MarketID, outcome.Name, i)))

		_, err := c.pool.Exec(ctx, `
			INSERT INTO sports_selections (id, market_id, name, odds_decimal, status, sort_order, odds88_selection_id)
			VALUES (gen_random_uuid(), $1, $2, $3, 'active', $4, $5)
			ON CONFLICT (odds88_selection_id) DO UPDATE SET
				name = EXCLUDED.name,
				odds_decimal = EXCLUDED.odds_decimal,
				updated_at = now()`,
			marketID, selName, oddsDecimal, i+1, odds88SelectionID)
		if err != nil {
			c.logger.Debug("odds api upsert selection", "name", selName, "error", err)
		}
	}

	return nil
}

// hashOddsID converts an Odds API string ID to a stable int64 for odds88_event_id column.
func hashOddsID(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
