package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/guard"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PluginService manages plugin dispatch with guard chain evaluation.
type PluginService struct {
	pool        *pgxpool.Pool
	rateLimiter *guard.RateLimiter
	circuitBkr  *guard.CircuitBreaker
	idempotency *guard.IdempotencyGuard
	logger      *slog.Logger
}

// NewPluginService creates a new PluginService with guard chain.
func NewPluginService(pool *pgxpool.Pool, logger *slog.Logger) *PluginService {
	return &PluginService{
		pool:        pool,
		rateLimiter: guard.NewRateLimiter(60, time.Minute),
		circuitBkr:  guard.NewCircuitBreaker(5, 30*time.Second),
		idempotency: guard.NewIdempotencyGuard(),
		logger:      logger,
	}
}

// DispatchInput is the request to dispatch a plugin.
type DispatchInput struct {
	PluginID       string      `json:"plugin_id"`
	Scope          string      `json:"scope"`
	Payload        interface{} `json:"payload"`
	PlayerID       *uuid.UUID  `json:"player_id,omitempty"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
}

// DispatchResult is the outcome of a plugin dispatch.
type DispatchResult struct {
	DispatchID uuid.UUID          `json:"dispatch_id"`
	Status     domain.DispatchStatus `json:"status"`
	Result     interface{}        `json:"result,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// Dispatch executes a plugin through the guard chain.
func (s *PluginService) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	// Guard chain: rate limit → circuit breaker → idempotency
	if result := s.rateLimiter.Check(ctx, input.PluginID); !result.Allowed {
		return &DispatchResult{
			Status: domain.DispatchFailed,
			Error:  result.Reason,
		}, nil
	}

	if result := s.circuitBkr.Check(ctx, input.PluginID); !result.Allowed {
		return &DispatchResult{
			Status: domain.DispatchFallback,
			Error:  result.Reason,
		}, nil
	}

	if result := s.idempotency.Check(ctx, input.IdempotencyKey); !result.Allowed {
		return &DispatchResult{
			Status: domain.DispatchFailed,
			Error:  result.Reason,
		}, nil
	}

	// Record dispatch
	var dispatchID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO plugin_dispatches (plugin_id, scope, payload, status, player_id, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		input.PluginID, input.Scope, input.Payload,
		string(domain.DispatchRequested), input.PlayerID, input.IdempotencyKey,
	).Scan(&dispatchID)
	if err != nil {
		s.circuitBkr.RecordFailure(input.PluginID)
		return nil, domain.ErrInternal("create dispatch", err)
	}

	// Simulate dispatch execution (actual plugin call would go here)
	s.circuitBkr.RecordSuccess(input.PluginID)

	_, err = s.pool.Exec(ctx, `
		UPDATE plugin_dispatches SET status = $2, updated_at = now() WHERE id = $1`,
		dispatchID, string(domain.DispatchCompleted))
	if err != nil {
		s.logger.Error("failed to update dispatch status", "dispatch_id", dispatchID, "error", err)
	}

	return &DispatchResult{
		DispatchID: dispatchID,
		Status:     domain.DispatchCompleted,
	}, nil
}

// ListDispatches returns recent dispatches for a plugin.
func (s *PluginService) ListDispatches(ctx context.Context, pluginID string) ([]domain.PluginDispatch, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, plugin_id, status, error, created_at
		FROM plugin_dispatches
		WHERE plugin_id = $1
		ORDER BY created_at DESC LIMIT 50`, pluginID)
	if err != nil {
		return nil, domain.ErrInternal("list dispatches", err)
	}
	defer rows.Close()

	var dispatches []domain.PluginDispatch
	for rows.Next() {
		var d domain.PluginDispatch
		var errStr *string
		if err := rows.Scan(&d.ID, &d.PluginID, &d.Status, &errStr, &d.CreatedAt); err != nil {
			return nil, domain.ErrInternal("scan dispatch", err)
		}
		if errStr != nil {
			d.Error = *errStr
		}
		dispatches = append(dispatches, d)
	}

	return dispatches, nil
}

// ListPlugins returns all registered plugins.
func (s *PluginService) ListPlugins(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, plugin_id, name, domain, risk_tier, active, created_at
		FROM plugins ORDER BY name ASC`)
	if err != nil {
		return nil, domain.ErrInternal("list plugins", err)
	}
	defer rows.Close()

	var plugins []map[string]interface{}
	for rows.Next() {
		var (
			id        uuid.UUID
			pluginID  string
			name      string
			dom       string
			riskTier  string
			active    bool
			createdAt time.Time
		)
		if err := rows.Scan(&id, &pluginID, &name, &dom, &riskTier, &active, &createdAt); err != nil {
			return nil, fmt.Errorf("scan plugin: %w", err)
		}
		plugins = append(plugins, map[string]interface{}{
			"id":         id,
			"plugin_id":  pluginID,
			"name":       name,
			"domain":     dom,
			"risk_tier":  riskTier,
			"active":     active,
			"created_at": createdAt,
		})
	}

	return plugins, nil
}
