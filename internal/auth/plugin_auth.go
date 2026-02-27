package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
)

// PluginAuthManager handles HMAC-SHA256 scoped tokens for plugins.
type PluginAuthManager struct {
	secret []byte
}

// NewPluginAuthManager creates a plugin auth manager.
func NewPluginAuthManager(secret string) *PluginAuthManager {
	return &PluginAuthManager{secret: []byte(secret)}
}

// GeneratePluginToken creates an HMAC-SHA256 scoped token.
// Format: base64(payload).base64(signature)
func (m *PluginAuthManager) GeneratePluginToken(pluginID string, scopes []string, riskTier domain.RiskTier) (string, error) {
	ttl := domain.RiskTierTTL(riskTier)
	now := time.Now()

	token := domain.PluginScopedToken{
		Sub:      pluginID,
		Scopes:   scopes,
		RiskTier: riskTier,
		Exp:      now.Add(ttl).Unix(),
		Iat:      now.Unix(),
		Jti:      uuid.New().String(),
	}

	payloadJSON, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("marshal plugin token: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sig := m.sign(payloadB64)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64, nil
}

// ValidatePluginToken verifies and decodes a plugin scoped token.
func (m *PluginAuthManager) ValidatePluginToken(tokenString string) (*domain.PluginScopedToken, error) {
	parts := splitToken(tokenString)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid plugin token format")
	}

	payloadB64 := parts[0]
	sigB64 := parts[1]

	// Verify signature
	expectedSig := m.sign(payloadB64)
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var token domain.PluginScopedToken
	if err := json.Unmarshal(payloadJSON, &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > token.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &token, nil
}

func (m *PluginAuthManager) sign(data string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func splitToken(s string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
