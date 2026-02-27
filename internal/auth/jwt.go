package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Realm identifies the JWT authentication realm.
type Realm string

const (
	RealmPlayer    Realm = "player"
	RealmAdmin     Realm = "admin"
	RealmAffiliate Realm = "affiliate"
)

// Claims holds the custom JWT claims for all 3 realms.
type Claims struct {
	jwt.RegisteredClaims
	Realm  Realm    `json:"realm"`
	Email  string   `json:"email,omitempty"`
	Role   string   `json:"role,omitempty"`   // admin realm: viewer, admin, superadmin
	Status string   `json:"status,omitempty"` // affiliate realm: active, suspended
}

// JWTManager handles token generation and validation for all 3 realms.
type JWTManager struct {
	secret          []byte
	playerExpiry    time.Duration
	adminExpiry     time.Duration
	affiliateExpiry time.Duration
}

// NewJWTManager creates a JWT manager with realm-specific expiry durations.
func NewJWTManager(secret string, playerExpiry, adminExpiry, affiliateExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		playerExpiry:    playerExpiry,
		adminExpiry:     adminExpiry,
		affiliateExpiry: affiliateExpiry,
	}
}

// GenerateToken creates a signed JWT for the given realm and subject.
func (m *JWTManager) GenerateToken(realm Realm, subjectID uuid.UUID, email, role, status string) (string, error) {
	var expiry time.Duration
	switch realm {
	case RealmPlayer:
		expiry = m.playerExpiry
	case RealmAdmin:
		expiry = m.adminExpiry
	case RealmAffiliate:
		expiry = m.affiliateExpiry
	default:
		return "", fmt.Errorf("unknown realm: %s", realm)
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subjectID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			ID:        uuid.New().String(),
		},
		Realm:  realm,
		Email:  email,
		Role:   role,
		Status: status,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateToken parses and validates a JWT, returning claims if valid.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateTokenForRealm validates a token and ensures it belongs to the expected realm.
func (m *JWTManager) ValidateTokenForRealm(tokenString string, expectedRealm Realm) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Realm != expectedRealm {
		return nil, fmt.Errorf("expected realm %s, got %s", expectedRealm, claims.Realm)
	}
	return claims, nil
}
