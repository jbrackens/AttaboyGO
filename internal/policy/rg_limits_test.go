package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateRgLimits_AllowsWithinLimits(t *testing.T) {
	policy := DefaultRgLimits()
	result := EvaluateRgLimits(policy, 50_000, "wallet_deposit", 0, 0)
	assert.True(t, result.Allowed)
}

func TestEvaluateRgLimits_BlocksSingleTransactionOverLimit(t *testing.T) {
	policy := DefaultRgLimits()
	result := EvaluateRgLimits(policy, 150_000, "wallet_deposit", 0, 0)
	assert.False(t, result.Allowed)
	assert.Equal(t, "single_transaction", result.BreachedLimit)
}

func TestEvaluateRgLimits_BlocksDailyDepositOverLimit(t *testing.T) {
	policy := DefaultRgLimits()
	// Already deposited 180_000, trying to deposit 30_000 more (total 210_000 > 200_000)
	result := EvaluateRgLimits(policy, 30_000, "wallet_deposit", 180_000, 0)
	assert.False(t, result.Allowed)
	assert.Equal(t, "daily_deposit", result.BreachedLimit)
}

func TestEvaluateRgLimits_BlocksDailyLossOverLimit(t *testing.T) {
	policy := DefaultRgLimits()
	// Already lost 140_000, trying to bet 20_000 more (total 160_000 > 150_000)
	result := EvaluateRgLimits(policy, 20_000, "bet", 0, 140_000)
	assert.False(t, result.Allowed)
	assert.Equal(t, "daily_loss", result.BreachedLimit)
}

func TestEvaluateRgLimits_AllowsBetWithinDailyLoss(t *testing.T) {
	policy := DefaultRgLimits()
	result := EvaluateRgLimits(policy, 50_000, "bet", 0, 50_000)
	assert.True(t, result.Allowed)
}

func TestEvaluateRgLimits_DailyDepositDoesNotApplyToBets(t *testing.T) {
	policy := DefaultRgLimits()
	// Daily deposit is high but this is a bet, not a deposit
	result := EvaluateRgLimits(policy, 50_000, "bet", 199_000, 0)
	assert.True(t, result.Allowed)
}
