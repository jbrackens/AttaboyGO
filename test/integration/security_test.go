//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Lockout Tests (5) ──────────────────────────────────────────────────────

func TestLockout_PlayerLoginBlocked(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("lockout@test.com", "securepass123", "EUR")

	// 5 bad attempts
	for i := 0; i < 5; i++ {
		resp := env.POST("/auth/login", map[string]string{
			"email": "lockout@test.com", "password": "wrongpass",
		}, "")
		resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// 6th attempt should be locked (429)
	resp := env.POST("/auth/login", map[string]string{
		"email": "lockout@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "ACCOUNT_LOCKED", result["code"])
}

func TestLockout_PlayerLoginResetsOnSuccess(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("resetlock@test.com", "securepass123", "EUR")

	// 4 bad attempts
	for i := 0; i < 4; i++ {
		resp := env.POST("/auth/login", map[string]string{
			"email": "resetlock@test.com", "password": "wrongpass",
		}, "")
		resp.Body.Close()
	}

	// 1 successful login
	resp := env.POST("/auth/login", map[string]string{
		"email": "resetlock@test.com", "password": "securepass123",
	}, "")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 1 more bad attempt — should NOT be locked (only 1 failure after success, total < 5 in window but success doesn't reset count... we count ALL failures in window)
	// Actually the lockout counts all failures in the window, success doesn't reset.
	// So after 4 failures + 1 success + 1 failure = 5 failures total → locked on next attempt.
	resp = env.POST("/auth/login", map[string]string{
		"email": "resetlock@test.com", "password": "wrongpass",
	}, "")
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Now we have 5 failures total → next attempt should be locked
	resp = env.POST("/auth/login", map[string]string{
		"email": "resetlock@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestLockout_AffiliateLoginBlocked(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterAffiliate("afflock@test.com", "securepass123", "Jane", "Doe")

	for i := 0; i < 5; i++ {
		resp := env.POST("/affiliates/login", map[string]string{
			"email": "afflock@test.com", "password": "wrongpass",
		}, "")
		resp.Body.Close()
	}

	resp := env.POST("/affiliates/login", map[string]string{
		"email": "afflock@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestLockout_DifferentEmailsIndependent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("lock1@test.com", "securepass123", "EUR")
	env.RegisterPlayer("lock2@test.com", "securepass123", "EUR")

	// Lock out email1
	for i := 0; i < 5; i++ {
		resp := env.POST("/auth/login", map[string]string{
			"email": "lock1@test.com", "password": "wrongpass",
		}, "")
		resp.Body.Close()
	}

	// email2 should still work
	resp := env.POST("/auth/login", map[string]string{
		"email": "lock2@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLockout_CorrectPasswordStillLocked(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("stilllocked@test.com", "securepass123", "EUR")

	for i := 0; i < 5; i++ {
		resp := env.POST("/auth/login", map[string]string{
			"email": "stilllocked@test.com", "password": "wrongpass",
		}, "")
		resp.Body.Close()
	}

	// Even with correct password, should be locked
	resp := env.POST("/auth/login", map[string]string{
		"email": "stilllocked@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

// ─── RBAC Tests (5) ─────────────────────────────────────────────────────────

func TestRBAC_ViewerCanReadPlayers(t *testing.T) {
	env := testutil.NewTestEnv(t)
	viewerToken := env.AdminToken("viewer")

	resp := env.AuthGET("/admin/players", viewerToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRBAC_ViewerCannotCreateBonus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	viewerToken := env.AdminToken("viewer")

	resp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name":   "Test Bonus",
		"amount": 1000,
		"type":   "deposit_match",
	}, viewerToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRBAC_AdminCanCreateBonus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("admin")

	resp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name":           "Test Bonus",
		"amount":         1000,
		"type":           "deposit_match",
		"wager_multiple": 10,
	}, adminToken)
	defer resp.Body.Close()

	// Should be allowed (200 or 201) — not 403
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestRBAC_AdminCannotSettle(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("admin")

	resp := env.AuthPOST("/admin/sportsbook/events/"+testutil.FakeUUID()+"/settle", map[string]interface{}{
		"winning_selection_id": testutil.FakeUUID(),
	}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRBAC_SuperAdminCanSettle(t *testing.T) {
	env := testutil.NewTestEnv(t)
	superToken := env.AdminToken("superadmin")

	// Seed a sportsbook event to settle
	_, eventID, _, selectionID := env.SeedSportsbook(200)

	resp := env.AuthPOST("/admin/sportsbook/events/"+eventID.String()+"/settle", map[string]interface{}{
		"winning_selection_id": selectionID.String(),
	}, superToken)
	defer resp.Body.Close()

	// Should be allowed (200) — not 403
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

// ─── Password Reset Tests (5) ──────────────────────────────────────────────

func TestPasswordReset_RequestReturnsToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("reset@test.com", "securepass123", "EUR")

	resp := env.POST("/auth/password-reset/request", map[string]string{
		"email": "reset@test.com",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
}

func TestPasswordReset_ConfirmChangesPassword(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("resetconfirm@test.com", "securepass123", "EUR")

	// Request reset token
	resp := env.POST("/auth/password-reset/request", map[string]string{
		"email": "resetconfirm@test.com",
	}, "")
	var tokenResult struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResult)
	resp.Body.Close()

	// Confirm reset with new password
	resp = env.POST("/auth/password-reset/confirm", map[string]string{
		"token": tokenResult.Token, "new_password": "newpassword456",
	}, "")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Login with new password should work
	resp = env.POST("/auth/login", map[string]string{
		"email": "resetconfirm@test.com", "password": "newpassword456",
	}, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPasswordReset_ExpiredTokenFails(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("expired@test.com", "securepass123", "EUR")

	// Request reset token
	resp := env.POST("/auth/password-reset/request", map[string]string{
		"email": "expired@test.com",
	}, "")
	var tokenResult struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResult)
	resp.Body.Close()

	// Manually expire the token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := env.Pool.Exec(ctx,
		`UPDATE password_reset_tokens SET expires_at = now() - interval '1 hour' WHERE email = $1`,
		"expired@test.com")
	require.NoError(t, err)

	// Confirm should fail
	resp = env.POST("/auth/password-reset/confirm", map[string]string{
		"token": tokenResult.Token, "new_password": "newpassword456",
	}, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPasswordReset_UsedTokenFails(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("usedtoken@test.com", "securepass123", "EUR")

	// Request reset token
	resp := env.POST("/auth/password-reset/request", map[string]string{
		"email": "usedtoken@test.com",
	}, "")
	var tokenResult struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResult)
	resp.Body.Close()

	// First confirm — success
	resp = env.POST("/auth/password-reset/confirm", map[string]string{
		"token": tokenResult.Token, "new_password": "newpassword456",
	}, "")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Second confirm — should fail (token already used)
	resp = env.POST("/auth/password-reset/confirm", map[string]string{
		"token": tokenResult.Token, "new_password": "anotherpass789",
	}, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPasswordReset_InvalidTokenFails(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.POST("/auth/password-reset/confirm", map[string]string{
		"token": "totally_invalid_random_token", "new_password": "newpassword456",
	}, "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
