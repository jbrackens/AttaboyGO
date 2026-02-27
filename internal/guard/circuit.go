package guard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/attaboy/platform/internal/domain"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements a per-plugin circuit breaker.
type CircuitBreaker struct {
	mu             sync.RWMutex
	circuits       map[string]*circuit
	failThreshold  int
	resetTimeout   time.Duration
	halfOpenMax    int
}

type circuit struct {
	state        CircuitState
	failures     int
	successes    int
	lastFailure  time.Time
}

// NewCircuitBreaker creates a circuit breaker with configurable thresholds.
func NewCircuitBreaker(failThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		circuits:      make(map[string]*circuit),
		failThreshold: failThreshold,
		resetTimeout:  resetTimeout,
		halfOpenMax:   1,
	}
}

// Check returns whether the circuit for the given key allows a request.
func (cb *CircuitBreaker) Check(_ context.Context, key string) domain.GuardResult {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	c, ok := cb.circuits[key]
	if !ok {
		cb.circuits[key] = &circuit{state: CircuitClosed}
		return domain.GuardResult{Allowed: true}
	}

	switch c.state {
	case CircuitOpen:
		if time.Since(c.lastFailure) > cb.resetTimeout {
			c.state = CircuitHalfOpen
			c.successes = 0
			return domain.GuardResult{Allowed: true}
		}
		return domain.GuardResult{
			Allowed: false,
			Reason:  fmt.Sprintf("circuit open for %s, resets in %s", key, cb.resetTimeout-time.Since(c.lastFailure)),
			Guard:   "circuit_breaker",
		}
	case CircuitHalfOpen:
		if c.successes >= cb.halfOpenMax {
			return domain.GuardResult{
				Allowed: false,
				Reason:  "circuit half-open, max probes reached",
				Guard:   "circuit_breaker",
			}
		}
		return domain.GuardResult{Allowed: true}
	default:
		return domain.GuardResult{Allowed: true}
	}
}

// RecordSuccess marks a successful execution for the given key.
func (cb *CircuitBreaker) RecordSuccess(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	c, ok := cb.circuits[key]
	if !ok {
		return
	}

	switch c.state {
	case CircuitHalfOpen:
		c.successes++
		if c.successes >= cb.halfOpenMax {
			c.state = CircuitClosed
			c.failures = 0
		}
	case CircuitClosed:
		c.failures = 0
	}
}

// RecordFailure marks a failed execution for the given key.
func (cb *CircuitBreaker) RecordFailure(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	c, ok := cb.circuits[key]
	if !ok {
		cb.circuits[key] = &circuit{state: CircuitClosed, failures: 1, lastFailure: time.Now()}
		return
	}

	c.failures++
	c.lastFailure = time.Now()

	if c.failures >= cb.failThreshold {
		c.state = CircuitOpen
	}
}
