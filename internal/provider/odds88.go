package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Odds88Connector manages the Odds88 delta and settlement feed connections.
type Odds88Connector struct {
	pool    *pgxpool.Pool
	baseURL string
	apiKey  string
	logger  *slog.Logger
	client  *http.Client
}

// NewOdds88Connector creates a new Odds88 feed connector.
func NewOdds88Connector(pool *pgxpool.Pool, baseURL, apiKey string, logger *slog.Logger) *Odds88Connector {
	return &Odds88Connector{
		pool:    pool,
		baseURL: baseURL,
		apiKey:  apiKey,
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Odds88DeltaEvent represents a single delta update from the Odds88 feed.
type Odds88DeltaEvent struct {
	EventID     int64   `json:"eventId"`
	MarketID    string  `json:"marketId"`
	SelectionID int64   `json:"selectionId"`
	Odds        float64 `json:"odds"`
	Status      string  `json:"status"`
	Revision    int64   `json:"revision"`
}

// Odds88SettlementEvent represents a settlement event from the Odds88 feed.
type Odds88SettlementEvent struct {
	EventID     int64  `json:"eventId"`
	MarketID    string `json:"marketId"`
	SelectionID int64  `json:"selectionId"`
	Result      string `json:"result"` // won, lost, void
	Revision    int64  `json:"revision"`
}

// StartDeltaFeed begins polling the Odds88 delta feed in a goroutine.
func (c *Odds88Connector) StartDeltaFeed(ctx context.Context) {
	c.logger.Info("odds88 delta feed starting")

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("odds88 delta feed stopped")
				c.updateFeedState(context.Background(), "delta", "disconnected", "")
				return
			case <-ticker.C:
				if err := c.pollDelta(ctx); err != nil {
					c.logger.Error("odds88 delta poll error", "error", err)
					c.updateFeedState(ctx, "delta", "error", err.Error())
				}
			}
		}
	}()
}

// StartSettlementFeed begins polling the Odds88 settlement feed.
func (c *Odds88Connector) StartSettlementFeed(ctx context.Context) {
	c.logger.Info("odds88 settlement feed starting")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("odds88 settlement feed stopped")
				c.updateFeedState(context.Background(), "settlement", "disconnected", "")
				return
			case <-ticker.C:
				if err := c.pollSettlement(ctx); err != nil {
					c.logger.Error("odds88 settlement poll error", "error", err)
					c.updateFeedState(ctx, "settlement", "error", err.Error())
				}
			}
		}
	}()
}

func (c *Odds88Connector) pollDelta(ctx context.Context) error {
	var lastRevision int64
	err := c.pool.QueryRow(ctx,
		`SELECT last_revision FROM odds88_feed_state WHERE feed_name = 'delta'`).
		Scan(&lastRevision)
	if err != nil {
		return fmt.Errorf("get delta revision: %w", err)
	}

	url := fmt.Sprintf("%s/delta?since=%d&apiKey=%s", c.baseURL, lastRevision, c.apiKey)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("fetch delta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delta feed returned %d", resp.StatusCode)
	}

	var events []Odds88DeltaEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return fmt.Errorf("decode delta: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	// Process events — update odds in sports_selections
	for _, e := range events {
		_, err := c.pool.Exec(ctx, `
			UPDATE sports_selections
			SET odds = $1, updated_at = now()
			WHERE odds88_selection_id = $2`,
			e.Odds, e.SelectionID)
		if err != nil {
			c.logger.Error("update selection odds", "selection_id", e.SelectionID, "error", err)
		}
	}

	maxRevision := events[len(events)-1].Revision
	c.updateFeedState(ctx, "delta", "connected", "")
	_, _ = c.pool.Exec(ctx,
		`UPDATE odds88_feed_state SET last_revision = $1, last_updated_at = now() WHERE feed_name = 'delta'`,
		maxRevision)

	c.logger.Debug("odds88 delta processed", "count", len(events), "revision", maxRevision)
	return nil
}

func (c *Odds88Connector) pollSettlement(ctx context.Context) error {
	var lastRevision int64
	err := c.pool.QueryRow(ctx,
		`SELECT last_revision FROM odds88_feed_state WHERE feed_name = 'settlement'`).
		Scan(&lastRevision)
	if err != nil {
		return fmt.Errorf("get settlement revision: %w", err)
	}

	url := fmt.Sprintf("%s/settlements?since=%d&apiKey=%s", c.baseURL, lastRevision, c.apiKey)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("fetch settlements: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("settlement feed returned %d", resp.StatusCode)
	}

	var events []Odds88SettlementEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return fmt.Errorf("decode settlement: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	c.logger.Info("odds88 settlements received", "count", len(events))

	maxRevision := events[len(events)-1].Revision
	c.updateFeedState(ctx, "settlement", "connected", "")
	_, _ = c.pool.Exec(ctx,
		`UPDATE odds88_feed_state SET last_revision = $1, last_updated_at = now() WHERE feed_name = 'settlement'`,
		maxRevision)

	return nil
}

func (c *Odds88Connector) updateFeedState(ctx context.Context, feed, status, errMsg string) {
	_, _ = c.pool.Exec(ctx, `
		UPDATE odds88_feed_state
		SET connection_status = $2, error_message = $3, last_updated_at = now()
		WHERE feed_name = $1`, feed, status, errMsg)
}
