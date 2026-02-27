package auth

import (
	"testing"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginTokenRoundtrip(t *testing.T) {
	mgr := NewPluginAuthManager("plugin-secret")

	token, err := mgr.GeneratePluginToken("my-plugin", []string{"read", "write"}, domain.RiskTierLow)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := mgr.ValidatePluginToken(token)
	require.NoError(t, err)
	assert.Equal(t, "my-plugin", parsed.Sub)
	assert.Equal(t, []string{"read", "write"}, parsed.Scopes)
	assert.Equal(t, domain.RiskTierLow, parsed.RiskTier)
}

func TestPluginTokenInvalidSignature(t *testing.T) {
	mgr1 := NewPluginAuthManager("secret-1")
	mgr2 := NewPluginAuthManager("secret-2")

	token, err := mgr1.GeneratePluginToken("plugin", nil, domain.RiskTierMedium)
	require.NoError(t, err)

	_, err = mgr2.ValidatePluginToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestPluginTokenRiskTierTTL(t *testing.T) {
	assert.Equal(t, 1*time.Hour, domain.RiskTierTTL(domain.RiskTierLow))
	assert.Equal(t, 30*time.Minute, domain.RiskTierTTL(domain.RiskTierMedium))
	assert.Equal(t, 5*time.Minute, domain.RiskTierTTL(domain.RiskTierHigh))
}
