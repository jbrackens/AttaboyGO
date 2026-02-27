package guard

import (
	"context"
	"sync"

	"github.com/attaboy/platform/internal/domain"
)

// IdempotencyGuard deduplicates requests by idempotency key.
type IdempotencyGuard struct {
	mu   sync.Mutex
	seen map[string]bool
}

// NewIdempotencyGuard creates a new in-memory idempotency guard.
func NewIdempotencyGuard() *IdempotencyGuard {
	return &IdempotencyGuard{
		seen: make(map[string]bool),
	}
}

// Check returns whether the given key has already been processed.
func (ig *IdempotencyGuard) Check(_ context.Context, key string) domain.GuardResult {
	if key == "" {
		return domain.GuardResult{Allowed: true}
	}

	ig.mu.Lock()
	defer ig.mu.Unlock()

	if ig.seen[key] {
		return domain.GuardResult{
			Allowed: false,
			Reason:  "duplicate request: idempotency key already processed",
			Guard:   "idempotency",
		}
	}

	ig.seen[key] = true
	return domain.GuardResult{Allowed: true}
}

// Remove deletes a key from the seen set (for retry scenarios).
func (ig *IdempotencyGuard) Remove(key string) {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	delete(ig.seen, key)
}
