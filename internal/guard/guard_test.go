package guard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		result := rl.Check(ctx, "test-key")
		assert.True(t, result.Allowed, "request %d should be allowed", i+1)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	ctx := context.Background()

	rl.Check(ctx, "test-key")
	rl.Check(ctx, "test-key")
	result := rl.Check(ctx, "test-key")

	assert.False(t, result.Allowed)
	assert.Equal(t, "rate_limiter", result.Guard)
}

func TestRateLimiter_SeparateKeys(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	ctx := context.Background()

	r1 := rl.Check(ctx, "key-a")
	r2 := rl.Check(ctx, "key-b")

	assert.True(t, r1.Allowed)
	assert.True(t, r2.Allowed)
}

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	ctx := context.Background()

	result := cb.Check(ctx, "plugin-a")
	assert.True(t, result.Allowed)
}

func TestCircuitBreaker_OpensOnThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second)
	ctx := context.Background()

	cb.Check(ctx, "plugin-a")
	cb.RecordFailure("plugin-a")
	cb.RecordFailure("plugin-a")

	result := cb.Check(ctx, "plugin-a")
	assert.False(t, result.Allowed)
	assert.Equal(t, "circuit_breaker", result.Guard)
}

func TestCircuitBreaker_SuccessResets(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second)
	ctx := context.Background()

	cb.Check(ctx, "plugin-a")
	cb.RecordFailure("plugin-a")
	cb.RecordSuccess("plugin-a")

	result := cb.Check(ctx, "plugin-a")
	assert.True(t, result.Allowed)
}

func TestIdempotencyGuard_AllowsFirst(t *testing.T) {
	ig := NewIdempotencyGuard()
	ctx := context.Background()

	result := ig.Check(ctx, "req-123")
	assert.True(t, result.Allowed)
}

func TestIdempotencyGuard_BlocksDuplicate(t *testing.T) {
	ig := NewIdempotencyGuard()
	ctx := context.Background()

	ig.Check(ctx, "req-123")
	result := ig.Check(ctx, "req-123")

	assert.False(t, result.Allowed)
	assert.Equal(t, "idempotency", result.Guard)
}

func TestIdempotencyGuard_EmptyKeyAllowed(t *testing.T) {
	ig := NewIdempotencyGuard()
	ctx := context.Background()

	r1 := ig.Check(ctx, "")
	r2 := ig.Check(ctx, "")

	assert.True(t, r1.Allowed)
	assert.True(t, r2.Allowed)
}

func TestIdempotencyGuard_RemoveAllowsRetry(t *testing.T) {
	ig := NewIdempotencyGuard()
	ctx := context.Background()

	ig.Check(ctx, "req-456")
	ig.Remove("req-456")

	result := ig.Check(ctx, "req-456")
	require.True(t, result.Allowed)
}
