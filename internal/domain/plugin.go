package domain

import (
	"time"

	"github.com/google/uuid"
)

// RiskTier controls HMAC token TTL for plugin scoped auth.
type RiskTier string

const (
	RiskTierLow    RiskTier = "low"    // 1h TTL
	RiskTierMedium RiskTier = "medium" // 30m TTL
	RiskTierHigh   RiskTier = "high"   // 5m TTL
)

// RiskTierTTL returns the token TTL for a given risk tier.
func RiskTierTTL(tier RiskTier) time.Duration {
	switch tier {
	case RiskTierHigh:
		return 5 * time.Minute
	case RiskTierMedium:
		return 30 * time.Minute
	default:
		return 1 * time.Hour
	}
}

// PluginScopedToken represents an HMAC-SHA256 scoped auth token.
type PluginScopedToken struct {
	Sub      string   `json:"sub"` // plugin ID
	Scopes   []string `json:"scopes"`
	RiskTier RiskTier `json:"riskTier"`
	Exp      int64    `json:"exp"`
	Iat      int64    `json:"iat"`
	Jti      string   `json:"jti"`
}

// DispatchStatus tracks plugin execution lifecycle.
type DispatchStatus string

const (
	DispatchRequested DispatchStatus = "requested"
	DispatchCompleted DispatchStatus = "completed"
	DispatchFailed    DispatchStatus = "failed"
	DispatchFallback  DispatchStatus = "fallback"
)

// ModerationAction is the result of plugin content moderation.
type ModerationAction string

const (
	ModerationBlocked ModerationAction = "blocked"
	ModerationFlagged ModerationAction = "flagged"
	ModerationPassed  ModerationAction = "passed"
)

// PluginDispatch records a plugin execution attempt.
type PluginDispatch struct {
	ID        uuid.UUID      `json:"id"`
	PluginID  string         `json:"plugin_id"`
	Status    DispatchStatus `json:"status"`
	Input     interface{}    `json:"input,omitempty"`
	Output    interface{}    `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration_ms"`
	CreatedAt time.Time      `json:"created_at"`
}

// FallbackStrategy defines plugin fallback behavior.
type FallbackStrategy string

const (
	FallbackNoop            FallbackStrategy = "noop"
	FallbackDefaultResponse FallbackStrategy = "default_response"
	FallbackQueueRetry      FallbackStrategy = "queue_retry"
	FallbackDelegate        FallbackStrategy = "delegate"
	FallbackCustom          FallbackStrategy = "custom"
)

// GuardResult is the outcome of a guard chain evaluation.
type GuardResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Guard   string `json:"guard,omitempty"` // which guard blocked
}
