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

// DomeConnector manages prediction market syncing from Dome (Polymarket, Kalshi).
type DomeConnector struct {
	pool    *pgxpool.Pool
	baseURL string
	apiKey  string
	logger  *slog.Logger
	client  *http.Client
}

// NewDomeConnector creates a new Dome prediction feed connector.
func NewDomeConnector(pool *pgxpool.Pool, baseURL, apiKey string, logger *slog.Logger) *DomeConnector {
	return &DomeConnector{
		pool:    pool,
		baseURL: baseURL,
		apiKey:  apiKey,
		logger:  logger,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// DomeMarket is a market from the Dome feed.
type DomeMarket struct {
	Platform    string    `json:"platform"` // polymarket, kalshi
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Outcomes    []DomeOutcome `json:"outcomes"`
}

// DomeOutcome is an outcome option within a Dome market.
type DomeOutcome struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Price float64 `json:"price"` // 0.0 - 1.0
}

// StartMarketSync begins periodic market syncing from Dome.
func (c *DomeConnector) StartMarketSync(ctx context.Context) {
	c.logger.Info("dome market sync starting")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
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
					c.updateFeedState(ctx, "polymarket-sync", "error", err.Error())
				}
			}
		}
	}()
}

func (c *DomeConnector) syncMarkets(ctx context.Context) error {
	url := fmt.Sprintf("%s/markets?apiKey=%s", c.baseURL, c.apiKey)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("fetch dome markets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dome API returned %d", resp.StatusCode)
	}

	var markets []DomeMarket
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return fmt.Errorf("decode dome markets: %w", err)
	}

	synced := 0
	for _, m := range markets {
		outcomesJSON, _ := json.Marshal(m.Outcomes)

		_, err := c.pool.Exec(ctx, `
			INSERT INTO prediction_markets (title, description, category, status, dome_platform, dome_market_slug, outcomes, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, (SELECT id FROM admin_users LIMIT 1))
			ON CONFLICT (dome_platform, dome_market_slug) WHERE dome_market_slug IS NOT NULL
			DO UPDATE SET status = $4, outcomes = $7, updated_at = now()`,
			m.Title, m.Description, m.Category, m.Status, m.Platform, m.Slug, outcomesJSON)
		if err != nil {
			c.logger.Error("upsert dome market", "slug", m.Slug, "error", err)
			continue
		}
		synced++
	}

	c.updateFeedState(ctx, "polymarket-sync", "active", "")
	c.logger.Debug("dome markets synced", "total", len(markets), "synced", synced)
	return nil
}

func (c *DomeConnector) updateFeedState(ctx context.Context, feed, status, errMsg string) {
	_, _ = c.pool.Exec(ctx, `
		UPDATE dome_feed_state
		SET status = $2, error_message = $3, last_sync_at = now(), updated_at = now()
		WHERE feed_name = $1`, feed, status, errMsg)
}
