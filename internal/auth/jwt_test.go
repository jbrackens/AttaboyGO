package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestJWTManager() *JWTManager {
	return NewJWTManager("test-secret-key", 24*time.Hour, 8*time.Hour, 12*time.Hour)
}

func TestGenerateAndValidatePlayerToken(t *testing.T) {
	mgr := newTestJWTManager()
	playerID := uuid.New()

	token, err := mgr.GenerateToken(RealmPlayer, playerID, "test@test.com", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := mgr.ValidateTokenForRealm(token, RealmPlayer)
	require.NoError(t, err)
	assert.Equal(t, playerID.String(), claims.Subject)
	assert.Equal(t, RealmPlayer, claims.Realm)
	assert.Equal(t, "test@test.com", claims.Email)
}

func TestGenerateAndValidateAdminToken(t *testing.T) {
	mgr := newTestJWTManager()
	adminID := uuid.New()

	token, err := mgr.GenerateToken(RealmAdmin, adminID, "admin@test.com", RoleSuperAdmin, "")
	require.NoError(t, err)

	claims, err := mgr.ValidateTokenForRealm(token, RealmAdmin)
	require.NoError(t, err)
	assert.Equal(t, RealmAdmin, claims.Realm)
	assert.Equal(t, RoleSuperAdmin, claims.Role)
}

func TestGenerateAndValidateAffiliateToken(t *testing.T) {
	mgr := newTestJWTManager()
	affID := uuid.New()

	token, err := mgr.GenerateToken(RealmAffiliate, affID, "aff@test.com", "", "active")
	require.NoError(t, err)

	claims, err := mgr.ValidateTokenForRealm(token, RealmAffiliate)
	require.NoError(t, err)
	assert.Equal(t, RealmAffiliate, claims.Realm)
	assert.Equal(t, "active", claims.Status)
}

func TestRealmMismatchRejected(t *testing.T) {
	mgr := newTestJWTManager()
	playerID := uuid.New()

	token, err := mgr.GenerateToken(RealmPlayer, playerID, "", "", "")
	require.NoError(t, err)

	_, err = mgr.ValidateTokenForRealm(token, RealmAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected realm admin")
}

func TestInvalidSecretRejected(t *testing.T) {
	mgr1 := NewJWTManager("secret-1", 24*time.Hour, 8*time.Hour, 12*time.Hour)
	mgr2 := NewJWTManager("secret-2", 24*time.Hour, 8*time.Hour, 12*time.Hour)

	token, err := mgr1.GenerateToken(RealmPlayer, uuid.New(), "", "", "")
	require.NoError(t, err)

	_, err = mgr2.ValidateToken(token)
	assert.Error(t, err)
}

func TestExpiredTokenRejected(t *testing.T) {
	mgr := NewJWTManager("secret", 1*time.Millisecond, 1*time.Millisecond, 1*time.Millisecond)

	token, err := mgr.GenerateToken(RealmPlayer, uuid.New(), "", "", "")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = mgr.ValidateToken(token)
	assert.Error(t, err)
}
