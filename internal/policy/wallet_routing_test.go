package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateWalletRoute_DefaultAllowsAll(t *testing.T) {
	policy := DefaultWalletRoutingPolicy()
	result := EvaluateWalletRoute(policy, "stripe", "wallet_deposit")
	assert.True(t, result.Allowed)
}

func TestEvaluateWalletRoute_BlockedSource(t *testing.T) {
	policy := WalletRoutingPolicy{BlockedSources: []string{"crypto"}}
	result := EvaluateWalletRoute(policy, "crypto", "wallet_deposit")
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "blocked")
}

func TestEvaluateWalletRoute_AllowedSourceWhitelist(t *testing.T) {
	policy := WalletRoutingPolicy{AllowedSources: []string{"stripe", "bank"}}

	result := EvaluateWalletRoute(policy, "stripe", "wallet_deposit")
	assert.True(t, result.Allowed)

	result = EvaluateWalletRoute(policy, "crypto", "wallet_deposit")
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "not in allowed list")
}

func TestEvaluateWalletRoute_AllowedTypeWhitelist(t *testing.T) {
	policy := WalletRoutingPolicy{AllowedTypes: []string{"wallet_deposit"}}

	result := EvaluateWalletRoute(policy, "stripe", "wallet_deposit")
	assert.True(t, result.Allowed)

	result = EvaluateWalletRoute(policy, "stripe", "wallet_withdrawal")
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "type not allowed")
}
