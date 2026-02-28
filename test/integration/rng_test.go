//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/stretchr/testify/assert"
)

// ─── RNG Tests (3) ────────────────────────────────────────────────────────

func TestRNG_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.POST("/rng/random", map[string]interface{}{
		"count": 1, "min": 1, "max": 100,
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRNG_EmptyBody(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("rngempty@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/rng/random", nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRNG_ValidationMaxLessThanMin(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("rngvalidate@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/rng/random", map[string]interface{}{
		"count": 1, "min": 100, "max": 1,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Slots Tests (2) ──────────────────────────────────────────────────────

func TestSlots_ListGamesRequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.GET("/slots/games")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSlots_SpinRequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.POST("/slots/spin", map[string]interface{}{
		"game_id": "test",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
