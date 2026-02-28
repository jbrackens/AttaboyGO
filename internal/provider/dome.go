package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Config ──

// DomeConfig holds Dome connector configuration.
type DomeConfig struct {
	BaseURL       string
	APIKey        string
	MinVolume     int           // minimum volume filter (default 50000)
	SyncInterval  time.Duration // market sync interval (default 5m)
	PriceInterval time.Duration // price update interval (default 30s)
	SettleInterval time.Duration // settlement check interval (default 60s)
	Platforms     []string      // polymarket, kalshi (default: polymarket)
}

// ── Rate Limiter ──

type rateLimiter struct {
	mu         sync.Mutex
	timestamps []int64
	maxQPS     int
	maxWindow  int
	windowMS   int64
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		maxQPS:    8,
		maxWindow: 90,
		windowMS:  10_000,
	}
}

func (rl *rateLimiter) wait() {
	for {
		rl.mu.Lock()
		now := time.Now().UnixMilli()

		// Purge old entries
		cutoff := now - rl.windowMS
		i := 0
		for i < len(rl.timestamps) && rl.timestamps[i] < cutoff {
			i++
		}
		rl.timestamps = rl.timestamps[i:]

		// Check window limit
		if len(rl.timestamps) >= rl.maxWindow {
			waitUntil := rl.timestamps[0] + rl.windowMS
			delay := waitUntil - now
			rl.mu.Unlock()
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
			continue
		}

		// Check per-second limit
		oneSecAgo := now - 1000
		recentCount := 0
		for _, t := range rl.timestamps {
			if t >= oneSecAgo {
				recentCount++
			}
		}
		if recentCount >= rl.maxQPS {
			rl.mu.Unlock()
			time.Sleep(150 * time.Millisecond)
			continue
		}

		rl.timestamps = append(rl.timestamps, now)
		rl.mu.Unlock()
		return
	}
}

// ── Dome API Types ──

type domePolymarketMarket struct {
	MarketSlug  string    `json:"market_slug"`
	EventSlug   string    `json:"event_slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ConditionID string    `json:"condition_id"`
	Status      string    `json:"status"`
	WinningSide *domeSide `json:"winning_side"` // null when open, {id, label} when resolved
	EndTime     *int64    `json:"end_time"`
	VolumeTotal float64   `json:"volume_total"`
	Tags        []string  `json:"tags"`
	Image       *string   `json:"image"`
	SideA       *domeSide `json:"side_a"`
	SideB       *domeSide `json:"side_b"`
}

type domeSide struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type domePolymarketResponse struct {
	Markets    []domePolymarketMarket `json:"markets"`
	Pagination domePagination         `json:"pagination"`
}

type domePagination struct {
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Total    int    `json:"total"`
	HasMore  bool   `json:"has_more"`
}

type domePriceResponse struct {
	Price  float64 `json:"price"`
	AtTime int64   `json:"at_time"`
}

// ── Category Mapping ──

var tagToCategory = map[string]string{
	"politics": "politics", "elections": "politics", "government": "politics",
	"us-politics": "politics", "world-politics": "politics", "geopolitics": "politics",
	"crypto": "crypto", "cryptocurrency": "crypto", "bitcoin": "crypto",
	"ethereum": "crypto", "defi": "crypto", "nft": "crypto", "blockchain": "crypto",
	"sports": "sports", "nfl": "sports", "nba": "sports", "mlb": "sports",
	"soccer": "sports", "football": "sports", "mma": "sports", "boxing": "sports",
	"entertainment": "entertainment", "movies": "entertainment", "music": "entertainment",
	"tv": "entertainment", "celebrity": "entertainment", "awards": "entertainment",
	"technology": "technology", "ai": "technology", "artificial-intelligence": "technology",
	"tech": "technology", "science": "technology", "space": "technology",
	"economics": "economics", "finance": "economics", "economy": "economics",
	"markets": "economics", "stocks": "economics", "interest-rates": "economics",
	"inflation": "economics", "fed": "economics",
}

func mapTagsToCategory(tags []string) string {
	for _, tag := range tags {
		normalised := strings.ToLower(strings.ReplaceAll(tag, " ", "-"))
		if cat, ok := tagToCategory[normalised]; ok {
			return cat
		}
	}
	return "general"
}

// ── Status / Odds Helpers ──

func mapDomeStatus(status string) string {
	switch strings.ToLower(status) {
	case "open", "active":
		return "open"
	default:
		return "closed"
	}
}

func probabilityToOdds(prob float64) float64 {
	if prob <= 0 {
		return 100.0
	}
	if prob >= 1 {
		return 1.01
	}
	raw := 1 / prob
	return math.Min(100.0, math.Max(1.01, math.Round(raw*1000)/1000))
}

// ── Outcome type for JSONB ──

type domeOutcome struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Odds        float64 `json:"odds"`
	DomeTokenID *string `json:"dome_token_id,omitempty"`
}

// ── DomeConnector ──

// DomeConnector manages prediction market syncing from Dome (Polymarket, Kalshi).
type DomeConnector struct {
	pool    *pgxpool.Pool
	cfg     DomeConfig
	logger  *slog.Logger
	client  *http.Client
	limiter *rateLimiter
}

// NewDomeConnector creates a new Dome prediction feed connector.
func NewDomeConnector(pool *pgxpool.Pool, baseURL, apiKey string, logger *slog.Logger) *DomeConnector {
	return &DomeConnector{
		pool: pool,
		cfg: DomeConfig{
			BaseURL:        baseURL,
			APIKey:         apiKey,
			MinVolume:      50000,
			SyncInterval:   5 * time.Minute,
			PriceInterval:  30 * time.Second,
			SettleInterval: 60 * time.Second,
			Platforms:      []string{"polymarket"},
		},
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
		limiter: newRateLimiter(),
	}
}

// StartMarketSync begins all three background jobs: market sync, price updater, settlement checker.
func (c *DomeConnector) StartMarketSync(ctx context.Context) {
	c.logger.Info("dome connector starting", "platforms", c.cfg.Platforms, "sync_interval", c.cfg.SyncInterval)

	// Run initial sync immediately
	go func() {
		if err := c.syncMarkets(ctx); err != nil {
			c.logger.Error("dome initial market sync error", "error", err)
		}
	}()

	// Market sync loop
	go func() {
		ticker := time.NewTicker(c.cfg.SyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("dome market sync stopped")
				c.updateFeedState(context.Background(), "polymarket-sync", "idle", "")
				return
			case <-ticker.C:
				if err := c.syncMarkets(ctx); err != nil {
					c.logger.Error("dome market sync error", "error", err)
				}
			}
		}
	}()

	// Price updater loop
	go func() {
		ticker := time.NewTicker(c.cfg.PriceInterval)
		defer ticker.Stop()
		cycleCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cycleCount++
				if err := c.updatePrices(ctx, cycleCount); err != nil {
					c.logger.Error("dome price update error", "error", err)
				}
			}
		}
	}()

	// Settlement checker loop
	go func() {
		ticker := time.NewTicker(c.cfg.SettleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.checkSettlements(ctx); err != nil {
					c.logger.Error("dome settlement check error", "error", err)
				}
			}
		}
	}()
}

// ── Rate-limited HTTP helper ──

func (c *DomeConnector) domeGet(ctx context.Context, path string) ([]byte, error) {
	c.limiter.wait()

	url := fmt.Sprintf("%s%s", c.cfg.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(math.Pow(2, float64(attempt+1))*500) * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("dome API returned %d", resp.StatusCode)
			delay := math.Pow(2, float64(attempt+1)) * 500
			c.logger.Warn("dome retry", "attempt", attempt+1, "status", resp.StatusCode, "delay_ms", delay)
			time.Sleep(time.Duration(delay) * time.Millisecond)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("dome API returned %d: %s", resp.StatusCode, string(body))
		}
		return body, nil
	}
	return nil, fmt.Errorf("dome API failed after 3 retries: %w", lastErr)
}

// ── Market Sync ──

func (c *DomeConnector) syncMarkets(ctx context.Context) error {
	c.updateFeedState(ctx, "polymarket-sync", "running", "")

	synced := 0
	hasMore := true
	offset := 0
	limit := 50

	for hasMore && offset <= 1000 {
		path := fmt.Sprintf("/v1/polymarket/markets?status=open&limit=%d&offset=%d", limit, offset)
		body, err := c.domeGet(ctx, path)
		if err != nil {
			c.updateFeedState(ctx, "polymarket-sync", "error", err.Error())
			return fmt.Errorf("fetch polymarket markets: %w", err)
		}

		var resp domePolymarketResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			c.updateFeedState(ctx, "polymarket-sync", "error", err.Error())
			return fmt.Errorf("decode polymarket markets: %w", err)
		}

		for _, m := range resp.Markets {
			// Volume filter
			if m.VolumeTotal < float64(c.cfg.MinVolume) {
				continue
			}

			if err := c.upsertPolymarketMarket(ctx, m); err != nil {
				c.logger.Error("upsert polymarket market", "slug", m.MarketSlug, "error", err)
				continue
			}
			synced++
		}

		hasMore = resp.Pagination.HasMore
		offset += limit
	}

	c.updateFeedState(ctx, "polymarket-sync", "idle", "")
	c.logger.Info("dome market sync complete", "synced", synced)
	return nil
}

func (c *DomeConnector) upsertPolymarketMarket(ctx context.Context, m domePolymarketMarket) error {
	status := mapDomeStatus(m.Status)
	category := mapTagsToCategory(m.Tags)

	sideALabel := "Yes"
	sideATokenID := ""
	if m.SideA != nil {
		sideALabel = m.SideA.Label
		sideATokenID = m.SideA.ID
	}
	sideBLabel := "No"
	if m.SideB != nil {
		sideBLabel = m.SideB.Label
	}

	outcomes := []domeOutcome{
		{ID: uuid.New().String(), Label: sideALabel, Odds: probabilityToOdds(0.5), DomeTokenID: &sideATokenID},
		{ID: uuid.New().String(), Label: sideBLabel, Odds: probabilityToOdds(0.5), DomeTokenID: nil},
	}
	outcomesJSON, _ := json.Marshal(outcomes)

	var winningSideLabel *string
	if m.WinningSide != nil {
		winningSideLabel = &m.WinningSide.Label
	}
	metadataJSON, _ := json.Marshal(map[string]any{
		"volume_total": m.VolumeTotal,
		"image":        m.Image,
		"end_time":     m.EndTime,
		"winning_side": winningSideLabel,
	})

	tagsJSON, _ := json.Marshal(m.Tags)

	var closeAt *time.Time
	if m.EndTime != nil {
		t := time.Unix(*m.EndTime, 0)
		closeAt = &t
	}

	// Upsert: on conflict update (but preserve settled/voided status and existing outcome IDs)
	_, err := c.pool.Exec(ctx, `
		INSERT INTO prediction_markets (
			title, description, category, status, close_at, outcomes,
			dome_platform, dome_market_slug, dome_condition_id, dome_event_slug,
			dome_metadata, dome_auto_settle, tags,
			created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			'polymarket', $7, $8, $9,
			$10, true, $11,
			(SELECT id FROM admin_users LIMIT 1)
		)
		ON CONFLICT (dome_platform, dome_market_slug) WHERE dome_market_slug IS NOT NULL
		DO UPDATE SET
			title = EXCLUDED.title,
			description = COALESCE(EXCLUDED.description, prediction_markets.description),
			category = EXCLUDED.category,
			status = CASE
				WHEN prediction_markets.status IN ('settled', 'voided') THEN prediction_markets.status
				ELSE EXCLUDED.status
			END,
			dome_condition_id = EXCLUDED.dome_condition_id,
			dome_event_slug = EXCLUDED.dome_event_slug,
			dome_metadata = EXCLUDED.dome_metadata,
			tags = EXCLUDED.tags,
			updated_at = now()`,
		m.Title, m.Description, category, status, closeAt, outcomesJSON,
		m.MarketSlug, m.ConditionID, m.EventSlug,
		metadataJSON, tagsJSON)
	return err
}

// ── Price Updater ──

func (c *DomeConnector) updatePrices(ctx context.Context, cycleCount int) error {
	c.updateFeedState(ctx, "price-updater", "running", "")

	rows, err := c.pool.Query(ctx, `
		SELECT id, dome_platform, dome_market_slug, outcomes
		FROM prediction_markets
		WHERE dome_platform IS NOT NULL AND status = 'open'`)
	if err != nil {
		c.updateFeedState(ctx, "price-updater", "error", err.Error())
		return err
	}
	defer rows.Close()

	type marketRow struct {
		ID       uuid.UUID
		Platform string
		Slug     string
		Outcomes json.RawMessage
	}
	var markets []marketRow
	for rows.Next() {
		var m marketRow
		if err := rows.Scan(&m.ID, &m.Platform, &m.Slug, &m.Outcomes); err != nil {
			continue
		}
		markets = append(markets, m)
	}

	// Get markets with active stakes
	stakedIDs := map[uuid.UUID]bool{}
	stakeRows, err := c.pool.Query(ctx, `
		SELECT DISTINCT market_id FROM prediction_stakes WHERE status = 'active'`)
	if err == nil {
		defer stakeRows.Close()
		for stakeRows.Next() {
			var id uuid.UUID
			if stakeRows.Scan(&id) == nil {
				stakedIDs[id] = true
			}
		}
	}

	updated := 0
	for _, market := range markets {
		// Skip non-staked markets on non-4th cycles
		if !stakedIDs[market.ID] && cycleCount%4 != 0 {
			continue
		}

		var outcomes []domeOutcome
		if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil || len(outcomes) < 2 {
			continue
		}

		var yesProb float64
		gotPrice := false

		if market.Platform == "polymarket" {
			// Find Yes outcome's dome_token_id
			for _, o := range outcomes {
				if o.DomeTokenID != nil && *o.DomeTokenID != "" && o.Label != "No" {
					path := fmt.Sprintf("/v1/polymarket/market-price/%s", *o.DomeTokenID)
					body, err := c.domeGet(ctx, path)
					if err != nil {
						c.logger.Warn("dome price fetch failed", "market_id", market.ID, "error", err)
						break
					}
					var priceResp domePriceResponse
					if json.Unmarshal(body, &priceResp) == nil {
						yesProb = priceResp.Price
						gotPrice = true
					}
					break
				}
			}
		}

		if !gotPrice {
			continue
		}

		noProb := 1 - yesProb
		yesOdds := probabilityToOdds(yesProb)
		noOdds := probabilityToOdds(noProb)

		// Check if odds changed >0.5%
		oldYesOdds := outcomes[0].Odds
		if oldYesOdds == 0 {
			oldYesOdds = 2.0
		}
		changePct := math.Abs(yesOdds-oldYesOdds) / oldYesOdds
		if changePct < 0.005 {
			continue
		}

		// Update outcomes
		outcomes[0].Odds = yesOdds
		if len(outcomes) > 1 {
			outcomes[1].Odds = noOdds
		}

		outJSON, _ := json.Marshal(outcomes)
		nowUnix := time.Now().Unix()
		_, err := c.pool.Exec(ctx, `
			UPDATE prediction_markets
			SET outcomes = $2, dome_last_price_at = $3, updated_at = now()
			WHERE id = $1`, market.ID, outJSON, nowUnix)
		if err == nil {
			updated++
			c.logger.Debug("dome price updated", "market_id", market.ID, "yes_odds", yesOdds, "no_odds", noOdds)
		}
	}

	c.updateFeedState(ctx, "price-updater", "idle", "")
	return nil
}

// ── Settlement Checker ──

func (c *DomeConnector) checkSettlements(ctx context.Context) error {
	c.updateFeedState(ctx, "settlement-checker", "running", "")

	rows, err := c.pool.Query(ctx, `
		SELECT id, dome_platform, dome_market_slug, outcomes, status
		FROM prediction_markets
		WHERE dome_platform IS NOT NULL
		  AND dome_auto_settle = true
		  AND status IN ('open', 'closed')`)
	if err != nil {
		c.updateFeedState(ctx, "settlement-checker", "error", err.Error())
		return err
	}
	defer rows.Close()

	type settleRow struct {
		ID       uuid.UUID
		Platform string
		Slug     string
		Outcomes json.RawMessage
		Status   string
	}
	var markets []settleRow
	for rows.Next() {
		var m settleRow
		if err := rows.Scan(&m.ID, &m.Platform, &m.Slug, &m.Outcomes, &m.Status); err != nil {
			continue
		}
		markets = append(markets, m)
	}

	settled := 0
	for _, market := range markets {
		var outcomes []domeOutcome
		if err := json.Unmarshal(market.Outcomes, &outcomes); err != nil || len(outcomes) < 2 {
			continue
		}

		var winningSide *domeSide

		if market.Platform == "polymarket" {
			// Re-fetch market from Dome
			path := fmt.Sprintf("/v1/polymarket/markets?market_slug=%s&limit=1", market.Slug)
			body, err := c.domeGet(ctx, path)
			if err != nil {
				continue
			}
			var resp domePolymarketResponse
			if json.Unmarshal(body, &resp) != nil || len(resp.Markets) == 0 {
				continue
			}
			dm := resp.Markets[0]
			winningSide = dm.WinningSide
		}

		if winningSide == nil {
			continue
		}

		// Map winning side to outcome by matching the winning side's ID/label to side_a (index 0) or side_b (index 1)
		var winningOutcomeID string
		// The winning_side object matches either side_a or side_b. side_a = outcomes[0], side_b = outcomes[1].
		for _, o := range outcomes {
			if strings.EqualFold(o.Label, winningSide.Label) {
				winningOutcomeID = o.ID
				break
			}
		}
		// Fallback: if label "Yes" matches first outcome
		if winningOutcomeID == "" && len(outcomes) > 0 {
			winningOutcomeID = outcomes[0].ID
		}

		if winningOutcomeID == "" {
			c.logger.Warn("dome: could not map winning side to outcome", "market_id", market.ID, "winning_label", winningSide.Label)
			continue
		}

		// Build attestation
		digestInput := fmt.Sprintf("dome:%s:%s:%s:%d", market.Platform, market.Slug, winningSide.Label, time.Now().UnixMilli())
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(digestInput)))

		attestation, _ := json.Marshal(map[string]string{
			"provider":      "dome",
			"attestationId": fmt.Sprintf("dome-%s-%s", market.Platform, uuid.New().String()[:8]),
			"digest":        digest,
			"issuedAt":      time.Now().UTC().Format(time.RFC3339),
		})

		winUUID, _ := uuid.Parse(winningOutcomeID)

		_, err := c.pool.Exec(ctx, `
			UPDATE prediction_markets
			SET status = 'settled', winning_outcome_id = $2, attestation = $3, updated_at = now()
			WHERE id = $1`, market.ID, winUUID, attestation)
		if err != nil {
			c.logger.Error("dome settle market", "market_id", market.ID, "error", err)
			continue
		}

		settled++
		c.logger.Info("dome market auto-settled", "market_id", market.ID, "slug", market.Slug, "winning_side", *winningSide)
	}

	c.updateFeedState(ctx, "settlement-checker", "idle", "")
	return nil
}

// ── Feed State ──

func (c *DomeConnector) updateFeedState(ctx context.Context, feed, status, errMsg string) {
	_, _ = c.pool.Exec(ctx, `
		UPDATE dome_feed_state
		SET status = $2, error_message = $3, last_sync_at = now(), updated_at = now()
		WHERE feed_name = $1`, feed, status, errMsg)
}
