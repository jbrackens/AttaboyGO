package guard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/attaboy/platform/internal/domain"
)

// RateLimiter implements a sliding window rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with the given limit per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

// Check returns a GuardResult indicating whether the key is within rate limits.
func (rl *RateLimiter) Check(_ context.Context, key string) domain.GuardResult {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove expired entries
	entries := rl.windows[key]
	valid := entries[:0]
	for _, t := range entries {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.windows[key] = valid
		return domain.GuardResult{
			Allowed: false,
			Reason:  fmt.Sprintf("rate limit exceeded: %d/%s", rl.limit, rl.window),
			Guard:   "rate_limiter",
		}
	}

	rl.windows[key] = append(valid, now)
	return domain.GuardResult{Allowed: true}
}
