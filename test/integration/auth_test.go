//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Registration Tests (10) ────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/register", map[string]string{
		"email": "newplayer@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		Token    string    `json:"token"`
		PlayerID uuid.UUID `json:"player_id"`
		Email    string    `json:"email"`
		Balance  struct {
			Balance      int64 `json:"balance"`
			BonusBalance int64 `json:"bonus_balance"`
		} `json:"balance"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
	assert.NotEqual(t, uuid.Nil, result.PlayerID)
	assert.Equal(t, int64(0), result.Balance.Balance)
	assert.Equal(t, int64(0), result.Balance.BonusBalance)
}

func TestRegister_CreatesThreeRows(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("threerows@test.com", "securepass123", "EUR")

	var authCount, playerCount, profileCount int
	env.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM auth_users WHERE id = $1", playerID).Scan(&authCount)
	env.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM v2_players WHERE id = $1", playerID).Scan(&playerCount)
	env.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM player_profiles WHERE player_id = $1", playerID).Scan(&profileCount)

	assert.Equal(t, 1, authCount)
	assert.Equal(t, 1, playerCount)
	assert.Equal(t, 1, profileCount)
}

func TestRegister_DefaultCurrency(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("defcur@test.com", "securepass123", "")

	var currency string
	env.Pool.QueryRow(t.Context(), "SELECT currency FROM v2_players WHERE id = $1", playerID).Scan(&currency)
	assert.Equal(t, "EUR", currency)
}

func TestRegister_CustomCurrency(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("usdplayer@test.com", "securepass123", "USD")

	var currency string
	env.Pool.QueryRow(t.Context(), "SELECT currency FROM v2_players WHERE id = $1", playerID).Scan(&currency)
	assert.Equal(t, "USD", currency)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("dup@test.com", "securepass123", "EUR")

	resp := env.POST("/auth/register", map[string]string{
		"email": "dup@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRegister_CaseInsensitiveEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("casetest@test.com", "securepass123", "EUR")

	resp := env.POST("/auth/register", map[string]string{
		"email": "CASETEST@TEST.COM", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRegister_InvalidEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/register", map[string]string{
		"email": "not-an-email", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRegister_EmptyEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/register", map[string]string{
		"email": "", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRegister_ShortPassword(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/register", map[string]string{
		"email": "shortpw@test.com", "password": "1234567",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRegister_EmptyBody(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/register", nil, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Login Tests (8) ────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("logintest@test.com", "securepass123", "EUR")

	resp := env.POST("/auth/login", map[string]string{
		"email": "logintest@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Token string `json:"token"`
		Email string `json:"email"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "logintest@test.com", result.Email)
}

func TestLogin_ReturnsCurrentBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("ballogin@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.POST("/auth/login", map[string]string{
		"email": "ballogin@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	var result struct {
		Balance struct {
			Balance int64 `json:"balance"`
		} `json:"balance"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, int64(5000), result.Balance.Balance)
}

func TestLogin_WrongPassword(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("wrongpw@test.com", "securepass123", "EUR")

	resp := env.POST("/auth/login", map[string]string{
		"email": "wrongpw@test.com", "password": "wrongpassword",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogin_NonexistentEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/login", map[string]string{
		"email": "noexist@test.com", "password": "securepass123",
	}, "")
	defer resp.Body.Close()

	// Should return same error as wrong password (no info leak)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogin_EmptyBody(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/auth/login", nil, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLogin_ValidJWT(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("jwttest@test.com", "securepass123", "EUR")

	// Parse the JWT to check claims
	parsed, err := jwt.ParseWithClaims(token, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(testutil.TestJWTSecret), nil
	})
	require.NoError(t, err)

	claims := parsed.Claims.(*auth.Claims)
	assert.Equal(t, auth.RealmPlayer, claims.Realm)
	assert.Equal(t, playerID.String(), claims.Subject)
}

func TestLogin_TokenWorksForProtectedRoute(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("protected@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/players/me", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin_TokenRejectedOnAdminRoute(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("noadmin@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/admin/players", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── JWT Middleware Tests (7) ───────────────────────────────────────────────

func TestPlayerRoute_NoToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/wallet/balance")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPlayerRoute_MalformedToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.AuthGET("/wallet/balance", "not.a.valid.jwt")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPlayerRoute_WrongRealmToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	// Admin token on player route
	adminToken := env.AdminToken("admin")
	resp := env.AuthGET("/wallet/balance", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminRoute_PlayerTokenRejected(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("notadmin@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/admin/players", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminRoute_ValidAdminToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealth_NoAuthRequired(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/health")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "healthy", result["status"])
}

func TestCORS_OptionsRequest(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.OPTIONS("/health")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.True(t, strings.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "GET"))
}
