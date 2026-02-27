//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Admin Auth Tests (4) ──────────────────────────────────────────────────

func TestAdminAuth_NoToken(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/admin/players")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminAuth_PlayerTokenRejected(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("adminreject@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/admin/players", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminAuth_ValidSuperadmin(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminAuth_ValidAdmin(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("admin")

	resp := env.AuthGET("/admin/players", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Player Management Tests (6) ──────────────────────────────────────────

func TestAdminPlayers_SearchEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminPlayers_FindByEmail(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("findme@test.com", "securepass123", "EUR")
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players?q=findme", adminToken)
	defer resp.Body.Close()

	var players []struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&players))
	assert.GreaterOrEqual(t, len(players), 1)
	assert.Equal(t, "findme@test.com", players[0].Email)
}

func TestAdminPlayers_NoMatch(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players?q=nonexistent999@nope.com", adminToken)
	defer resp.Body.Close()

	var players []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&players))
	assert.Empty(t, players)
}

func TestAdminPlayers_DetailFound(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("detail@test.com", "securepass123", "EUR")
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players/"+playerID.String(), adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Player  map[string]interface{} `json:"player"`
		Profile map[string]interface{} `json:"profile"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotNil(t, result.Player)
	assert.NotNil(t, result.Profile)
}

func TestAdminPlayers_Detail404(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/players/"+uuid.New().String(), adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAdminPlayers_UpdateStatus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("statusup@test.com", "securepass123", "EUR")
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthPATCH("/admin/players/"+playerID.String()+"/status",
		map[string]string{"account_status": "suspended"}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify in DB
	var status string
	env.Pool.QueryRow(t.Context(),
		"SELECT account_status FROM player_profiles WHERE player_id = $1", playerID).Scan(&status)
	assert.Equal(t, "suspended", status)
}

// ─── Bonus CRUD Tests (6) ─────────────────────────────────────────────────

func TestAdminBonuses_ListEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/bonuses", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminBonuses_Create(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "Welcome Bonus", "code": "WELCOME100",
		"wagering_multiplier": 30, "min_deposit": 1000, "max_bonus": 10000,
		"days_until_expiry": 30,
	}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestAdminBonuses_DuplicateCode(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "First Bonus", "code": "UNIQUE123",
		"wagering_multiplier": 30, "min_deposit": 1000, "max_bonus": 10000,
	}, adminToken)

	resp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "Second Bonus", "code": "UNIQUE123",
		"wagering_multiplier": 30, "min_deposit": 1000, "max_bonus": 10000,
	}, adminToken)
	defer resp.Body.Close()

	// Should fail due to unique code constraint
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestAdminBonuses_ListReturnsCreated(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "Test Bonus", "code": "TESTLIST",
		"wagering_multiplier": 20, "min_deposit": 500, "max_bonus": 5000,
	}, adminToken)

	resp := env.AuthGET("/admin/bonuses", adminToken)
	defer resp.Body.Close()

	var bonuses []struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bonuses))
	assert.GreaterOrEqual(t, len(bonuses), 1)

	found := false
	for _, b := range bonuses {
		if b.Code == "TESTLIST" {
			found = true
			break
		}
	}
	assert.True(t, found, "created bonus should appear in list")
}

func TestAdminBonuses_Deactivate(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	createResp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "Deactivate Me", "code": "DEACT1",
		"wagering_multiplier": 20, "min_deposit": 500, "max_bonus": 5000,
	}, adminToken)
	var created struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	resp := env.AuthPATCH("/admin/bonuses/"+created.ID+"/status",
		map[string]bool{"active": false}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var active bool
	env.Pool.QueryRow(t.Context(), "SELECT active FROM bonuses WHERE id = $1", created.ID).Scan(&active)
	assert.False(t, active)
}

func TestAdminBonuses_Reactivate(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	createResp := env.AuthPOST("/admin/bonuses", map[string]interface{}{
		"name": "Reactivate Me", "code": "REACT1",
		"wagering_multiplier": 20, "min_deposit": 500, "max_bonus": 5000,
	}, adminToken)
	var created struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Deactivate first
	env.AuthPATCH("/admin/bonuses/"+created.ID+"/status",
		map[string]bool{"active": false}, adminToken)

	// Reactivate
	resp := env.AuthPATCH("/admin/bonuses/"+created.ID+"/status",
		map[string]bool{"active": true}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var active bool
	env.Pool.QueryRow(t.Context(), "SELECT active FROM bonuses WHERE id = $1", created.ID).Scan(&active)
	assert.True(t, active)
}

// ─── Sportsbook Admin Tests (6) ───────────────────────────────────────────

func TestAdminSportsbook_ListEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	// Need at least one sport for the JOIN to work
	resp := env.AuthGET("/admin/sportsbook/events", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminSportsbook_CreateEvent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")
	sportID, _, _, _ := env.SeedSportsbook(200)

	resp := env.AuthPOST("/admin/sportsbook/events", map[string]interface{}{
		"sport_id":  sportID,
		"league":    "La Liga",
		"home_team": "Barcelona",
		"away_team": "Real Madrid",
		"start_time": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct{ ID string `json:"id"` }
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestAdminSportsbook_ListReturnsCreated(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")
	env.SeedSportsbook(200)

	resp := env.AuthGET("/admin/sportsbook/events", adminToken)
	defer resp.Body.Close()

	var events []struct {
		HomeTeam string `json:"home_team"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&events))
	assert.GreaterOrEqual(t, len(events), 1)
}

func TestAdminSportsbook_UpdateStatus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")
	_, eventID, _, _ := env.SeedSportsbook(200)

	resp := env.AuthPATCH("/admin/sportsbook/events/"+eventID.String()+"/status",
		map[string]interface{}{"status": "live"}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status string
	env.Pool.QueryRow(t.Context(), "SELECT status FROM sports_events WHERE id = $1", eventID).Scan(&status)
	assert.Equal(t, "live", status)
}

func TestAdminSportsbook_UpdateScore(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")
	_, eventID, _, _ := env.SeedSportsbook(200)

	scoreHome := 2
	scoreAway := 1
	resp := env.AuthPATCH("/admin/sportsbook/events/"+eventID.String()+"/status",
		map[string]interface{}{"status": "finished", "score_home": scoreHome, "score_away": scoreAway}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var sh, sa int
	env.Pool.QueryRow(t.Context(), "SELECT score_home, score_away FROM sports_events WHERE id = $1", eventID).Scan(&sh, &sa)
	assert.Equal(t, 2, sh)
	assert.Equal(t, 1, sa)
}

func TestAdminSportsbook_RequiresSport(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	// Try creating an event with a nonexistent sport_id
	resp := env.AuthPOST("/admin/sportsbook/events", map[string]interface{}{
		"sport_id":   uuid.New(),
		"league":     "Test",
		"home_team":  "A",
		"away_team":  "B",
		"start_time": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}, adminToken)
	defer resp.Body.Close()

	// Should fail due to FK constraint
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ─── Reports Tests (4) ────────────────────────────────────────────────────

func TestAdminReports_DashboardEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/reports/dashboard", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats struct {
		TotalPlayers int `json:"total_players"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	assert.Equal(t, 0, stats.TotalPlayers)
}

func TestAdminReports_DashboardWithData(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.RegisterPlayer("report1@test.com", "securepass123", "EUR")
	env.RegisterPlayer("report2@test.com", "securepass123", "EUR")
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/reports/dashboard", adminToken)
	defer resp.Body.Close()

	var stats struct {
		TotalPlayers int `json:"total_players"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	assert.Equal(t, 2, stats.TotalPlayers)
}

func TestAdminReports_TransactionReportEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/reports/transactions", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminReports_TransactionReportWithData(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("txreport@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/reports/transactions", adminToken)
	defer resp.Body.Close()

	var summaries []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&summaries))
	assert.GreaterOrEqual(t, len(summaries), 1)
}

// ─── Quest Admin Tests (4) ────────────────────────────────────────────────

func TestAdminQuests_ListEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthGET("/admin/quests", adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminQuests_Create(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	resp := env.AuthPOST("/admin/quests", map[string]interface{}{
		"name": "Daily Deposit", "description": "Make a deposit today",
		"type": "standard", "target_progress": 1, "reward_amount": 500,
		"reward_currency": "EUR", "min_score": 0,
		"cooldown_minutes": 0, "daily_budget_minor": 250000,
	}, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestAdminQuests_Toggle(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	questID := env.SeedQuest("Toggle Quest", 5, 1000)

	// Quest starts active, toggle should deactivate
	resp := env.AuthPATCH("/admin/quests/"+questID.String()+"/toggle", nil, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var active bool
	env.Pool.QueryRow(t.Context(), "SELECT active FROM quests WHERE id = $1", questID).Scan(&active)
	assert.False(t, active)
}

func TestAdminQuests_FieldValidation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	adminToken := env.AdminToken("superadmin")

	// Empty body
	resp := env.AuthPOST("/admin/quests", nil, adminToken)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
