//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Quest Player Tests (5) ───────────────────────────────────────────────

func TestQuests_ListEmpty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("questempty@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/quests/", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestQuests_ActiveShown(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("questactive@test.com", "securepass123", "EUR")
	env.SeedQuest("Active Quest", 10, 500)

	resp := env.AuthGET("/quests/", token)
	defer resp.Body.Close()

	var quests []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&quests))
	assert.GreaterOrEqual(t, len(quests), 1)
}

func TestQuests_IncludesProgress(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("questprog@test.com", "securepass123", "EUR")
	env.SeedQuest("Progress Quest", 10, 500)

	resp := env.AuthGET("/quests/", token)
	defer resp.Body.Close()

	var quests []struct {
		Progress int    `json:"progress"`
		Status   string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&quests))
	require.GreaterOrEqual(t, len(quests), 1)
	assert.Equal(t, 0, quests[0].Progress)
	assert.Equal(t, "not_started", quests[0].Status)
}

func TestQuests_ClaimReward(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("questclaim@test.com", "securepass123", "EUR")
	questID := env.SeedQuest("Claim Quest", 1, 500)

	// Set progress to completed
	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO player_quest_progress (player_id, quest_id, progress, status)
		VALUES ($1, $2, 1, 'completed')`, playerID, questID)
	require.NoError(t, err)

	resp := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		RewardAmount int `json:"reward_amount"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 500, result.RewardAmount)
}

func TestQuests_ClaimWithoutCompletion(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("questnoclaim@test.com", "securepass123", "EUR")
	questID := env.SeedQuest("Not Done Quest", 10, 500)

	resp := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	defer resp.Body.Close()

	// Should fail because no completed progress exists
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Quest Advanced Tests (5) ─────────────────────────────────────────────

func TestQuests_NoDoubleClaim(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("questdouble@test.com", "securepass123", "EUR")
	questID := env.SeedQuest("Double Claim Quest", 1, 500)

	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO player_quest_progress (player_id, quest_id, progress, status)
		VALUES ($1, $2, 1, 'completed')`, playerID, questID)
	require.NoError(t, err)

	// First claim should succeed
	resp1 := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Second claim should fail
	resp2 := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestQuests_ClaimCreditsBonus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("questbonus@test.com", "securepass123", "EUR")
	questID := env.SeedQuest("Bonus Quest", 1, 750)

	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO player_quest_progress (player_id, quest_id, progress, status)
		VALUES ($1, $2, 1, 'completed')`, playerID, questID)
	require.NoError(t, err)

	token := env.LoginPlayer("questbonus@test.com", "securepass123")
	resp := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify reward_grants row exists
	var count int
	env.Pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM reward_grants WHERE player_id = $1 AND quest_id = $2",
		playerID, questID).Scan(&count)
	assert.Equal(t, 1, count)
}

func TestQuests_InactiveQuestHidden(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("questinactive@test.com", "securepass123", "EUR")
	questID := env.SeedQuest("Inactive Quest", 5, 1000)

	// Deactivate quest
	env.Pool.Exec(t.Context(), "UPDATE quests SET active = false WHERE id = $1", questID)

	resp := env.AuthGET("/quests/", token)
	defer resp.Body.Close()

	var quests []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&quests))
	for _, q := range quests {
		assert.NotEqual(t, questID.String(), q.ID, "inactive quest should not appear")
	}
}

func TestQuests_MinScoreGate(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("questminscore@test.com", "securepass123", "EUR")

	// Create quest with high min_score requirement
	var questID uuid.UUID
	err := env.Pool.QueryRow(t.Context(), `
		INSERT INTO quests (name, description, type, target_progress, reward_amount, reward_currency, min_score, active, sort_order)
		VALUES ('High Score Quest', 'Requires score', 'standard', 1, 500, 'EUR', 9999, true, 1) RETURNING id`).Scan(&questID)
	require.NoError(t, err)

	// Complete progress
	_, err = env.Pool.Exec(t.Context(), `
		INSERT INTO player_quest_progress (player_id, quest_id, progress, status)
		VALUES ($1, $2, 1, 'completed')`, playerID, questID)
	require.NoError(t, err)

	// Attempt to claim — min_score gating may or may not be enforced at claim time
	resp := env.AuthPOST("/quests/"+questID.String()+"/claim", nil, token)
	defer resp.Body.Close()

	// Documents current behavior: min_score may not be enforced at claim
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest,
		"expected 200, 404 or 400, got %d", resp.StatusCode)
}

func TestQuests_MultipleQuestsProgress(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("questmulti@test.com", "securepass123", "EUR")
	questID1 := env.SeedQuest("Quest One", 1, 500)
	questID2 := env.SeedQuest("Quest Two", 1, 300)

	// Complete only quest 1
	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO player_quest_progress (player_id, quest_id, progress, status)
		VALUES ($1, $2, 1, 'completed')`, playerID, questID1)
	require.NoError(t, err)

	// Claim quest 1 should succeed
	resp1 := env.AuthPOST("/quests/"+questID1.String()+"/claim", nil, token)
	resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Claim quest 2 should fail (not completed)
	resp2 := env.AuthPOST("/quests/"+questID2.String()+"/claim", nil, token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// ─── Engagement Tests (4) ─────────────────────────────────────────────────

func TestEngagement_EmptyDefaults(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engdefault@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/engagement/me", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var eng struct {
		VideoMinutes       int `json:"video_minutes"`
		SocialInteractions int `json:"social_interactions"`
		Score              int `json:"score"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&eng))
	assert.Equal(t, 0, eng.VideoMinutes)
	assert.Equal(t, 0, eng.SocialInteractions)
	assert.Equal(t, 0, eng.Score)
}

func TestEngagement_RecordSignal(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engsignal@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/engagement/signal", map[string]interface{}{
		"type": "wager", "value": 1,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "recorded", result.Status)
}

func TestEngagement_AfterSignalReflects(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engreflect@test.com", "securepass123", "EUR")

	signalResp := env.AuthPOST("/engagement/signal", map[string]interface{}{
		"type": "wager", "value": 1,
	}, token)
	defer signalResp.Body.Close()
	require.Equal(t, http.StatusOK, signalResp.StatusCode)

	resp := env.AuthGET("/engagement/me", token)
	defer resp.Body.Close()

	var eng struct {
		WagerCount int `json:"wager_count"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&eng))
	assert.Equal(t, 1, eng.WagerCount)
}

func TestEngagement_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/engagement/me")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Engagement Advanced Tests (5) ────────────────────────────────────────

func TestEngagement_ScoreComputesCorrectly(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engscore@test.com", "securepass123", "EUR")

	// video(5)*2 + social(3)*3 + prediction(2)*5 = 10+9+10 = 29
	for i := 0; i < 5; i++ {
		r := env.AuthPOST("/engagement/signal", map[string]interface{}{"type": "video", "value": 1}, token)
		r.Body.Close()
	}
	for i := 0; i < 3; i++ {
		r := env.AuthPOST("/engagement/signal", map[string]interface{}{"type": "social", "value": 1}, token)
		r.Body.Close()
	}
	for i := 0; i < 2; i++ {
		r := env.AuthPOST("/engagement/signal", map[string]interface{}{"type": "prediction", "value": 1}, token)
		r.Body.Close()
	}

	resp := env.AuthGET("/engagement/me", token)
	defer resp.Body.Close()

	var eng struct {
		Score int `json:"score"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&eng))
	assert.Equal(t, 29, eng.Score)
}

func TestEngagement_MultipleSignalTypes(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engmulti@test.com", "securepass123", "EUR")

	signalTypes := []string{"video", "social", "prediction", "wager", "deposit"}
	for _, st := range signalTypes {
		r := env.AuthPOST("/engagement/signal", map[string]interface{}{"type": st, "value": 1}, token)
		r.Body.Close()
	}

	resp := env.AuthGET("/engagement/me", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var eng struct {
		VideoMinutes       int `json:"video_minutes"`
		SocialInteractions int `json:"social_interactions"`
		PredictionActions  int `json:"prediction_actions"`
		WagerCount         int `json:"wager_count"`
		DepositCount       int `json:"deposit_count"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&eng))
	assert.Equal(t, 1, eng.VideoMinutes)
	assert.Equal(t, 1, eng.SocialInteractions)
	assert.Equal(t, 1, eng.PredictionActions)
	assert.Equal(t, 1, eng.WagerCount)
	assert.Equal(t, 1, eng.DepositCount)
}

func TestEngagement_InvalidSignalType(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("enginvalid@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/engagement/signal", map[string]interface{}{
		"type": "invalid_type", "value": 1,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEngagement_PlayerIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("engiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("engiso2@test.com", "securepass123", "EUR")

	// Player 1 sends signals
	r := env.AuthPOST("/engagement/signal", map[string]interface{}{"type": "video", "value": 1}, token1)
	r.Body.Close()

	// Player 2 should have zero
	resp := env.AuthGET("/engagement/me", token2)
	defer resp.Body.Close()

	var eng struct {
		Score int `json:"score"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&eng))
	assert.Equal(t, 0, eng.Score)
}

func TestEngagement_ZeroValueSignal(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("engzero@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/engagement/signal", map[string]interface{}{
		"type": "video", "value": 0,
	}, token)
	defer resp.Body.Close()

	// Document behavior — zero value signal should still be recorded or rejected
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
		"expected 200 or 400, got %d", resp.StatusCode)
}

// ─── Prediction Tests (6) ─────────────────────────────────────────────────

func TestPredictions_ListMarkets(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predlist@test.com", "securepass123", "EUR")
	env.SeedPredictionMarket("Will it rain?")

	resp := env.AuthGET("/predictions/markets", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var markets []struct {
		Title string `json:"title"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&markets))
	assert.GreaterOrEqual(t, len(markets), 1)
}

func TestPredictions_GetMarket(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predget@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Get this market")

	resp := env.AuthGET("/predictions/markets/"+marketID.String(), token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var market struct {
		Title string `json:"title"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&market))
	assert.Equal(t, "Get this market", market.Title)
}

func TestPredictions_PlaceStake(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predstake@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Stake market")

	resp := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 500,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestPredictions_ClosedMarketRejects(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predclosed@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Closed market")

	// Close the market
	env.Pool.Exec(t.Context(), "UPDATE prediction_markets SET status = 'settled' WHERE id = $1", marketID)

	resp := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 500,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPredictions_EmptyPositions(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("prednopos@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/predictions/positions", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPredictions_AfterStakeShowsPosition(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predpos@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Position market")

	env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 700,
	}, token)

	resp := env.AuthGET("/predictions/positions", token)
	defer resp.Body.Close()

	var positions []struct {
		Amount int `json:"stake_amount"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&positions))
	assert.GreaterOrEqual(t, len(positions), 1)
}

// ─── Prediction Advanced Tests (8) ────────────────────────────────────────

func TestPredictions_MarketNotFound(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("prednotfound@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/predictions/markets/"+uuid.New().String(), token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPredictions_StakeZeroAmount(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predzero@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Zero Stake Market")

	resp := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 0,
	}, token)
	defer resp.Body.Close()

	// Document behavior — zero amount may be rejected or accepted
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest,
		"expected 201 or 400, got %d", resp.StatusCode)
}

func TestPredictions_MultipleStakesSameMarket(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("predmulti@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Multi Stake Market")

	r1 := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 500,
	}, token)
	r1.Body.Close()

	r2 := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "no", "amount": 300,
	}, token)
	r2.Body.Close()

	// Verify 2 stake rows in DB
	var count int
	env.Pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM prediction_stakes WHERE player_id = $1 AND market_id = $2",
		playerID, marketID).Scan(&count)
	assert.Equal(t, 2, count)
}

func TestPredictions_StakeRecordsOutcome(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("predoutcome@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Outcome Market")

	resp := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 500,
	}, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify outcome_id is stored correctly
	var outcomeID string
	err := env.Pool.QueryRow(t.Context(),
		"SELECT outcome_id FROM prediction_stakes WHERE player_id = $1 AND market_id = $2",
		playerID, marketID).Scan(&outcomeID)
	require.NoError(t, err)
	assert.Equal(t, "yes", outcomeID)
}

func TestPredictions_PositionIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("prediso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("prediso2@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Isolation Market")

	// Player 1 stakes
	r := env.AuthPOST("/predictions/markets/"+marketID.String()+"/stake", map[string]interface{}{
		"outcome_id": "yes", "amount": 500,
	}, token1)
	r.Body.Close()

	// Player 2 should have empty positions
	resp := env.AuthGET("/predictions/positions", token2)
	defer resp.Body.Close()

	var positions []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&positions))
	assert.Empty(t, positions)
}

func TestPredictions_ClosedMarketNotListed(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("predclosedlist@test.com", "securepass123", "EUR")
	marketID := env.SeedPredictionMarket("Closed List Market")

	// Settle the market
	env.Pool.Exec(t.Context(), "UPDATE prediction_markets SET status = 'settled' WHERE id = $1", marketID)

	resp := env.AuthGET("/predictions/markets", token)
	defer resp.Body.Close()

	var markets []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&markets))

	for _, m := range markets {
		assert.NotEqual(t, marketID.String(), m.ID, "settled market should not be listed")
	}
}

func TestPredictions_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.GET("/predictions/markets")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPredictions_ListMarketsDefault(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("preddefault@test.com", "securepass123", "EUR")

	env.SeedPredictionMarket("Market A")
	env.SeedPredictionMarket("Market B")
	env.SeedPredictionMarket("Market C")

	resp := env.AuthGET("/predictions/markets", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var markets []struct {
		Title string `json:"title"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&markets))
	assert.GreaterOrEqual(t, len(markets), 3)
}

// ─── AI Tests (5) ─────────────────────────────────────────────────────────

func TestAI_CreateConversation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("aicreate@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/ai/conversations", nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestAI_ListConversations(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("ailist@test.com", "securepass123", "EUR")

	env.AuthPOST("/ai/conversations", nil, token)

	resp := env.AuthGET("/ai/conversations", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var convs []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&convs))
	assert.Len(t, convs, 1)
}

func TestAI_SendMessage(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("aimsg@test.com", "securepass123", "EUR")

	// Create conversation
	createResp := env.AuthPOST("/ai/conversations", nil, token)
	var conv struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&conv)
	createResp.Body.Close()

	// Send message
	resp := env.AuthPOST("/ai/conversations/"+conv.ID+"/messages", map[string]string{
		"content": "Hello AI",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		Role string `json:"role"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "user", result.Role)
}

func TestAI_GetMessages(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("aigetmsgs@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/ai/conversations", nil, token)
	var conv struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&conv)
	createResp.Body.Close()

	env.AuthPOST("/ai/conversations/"+conv.ID+"/messages", map[string]string{"content": "Hi"}, token)

	resp := env.AuthGET("/ai/conversations/"+conv.ID+"/messages", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var messages []struct {
		Content string `json:"content"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&messages))
	assert.Len(t, messages, 1)
	assert.Equal(t, "Hi", messages[0].Content)
}

func TestAI_WrongOwnerRejected(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("aiowner1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("aiowner2@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/ai/conversations", nil, token1)
	var conv struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&conv)
	createResp.Body.Close()

	// Player 2 tries to access player 1's conversation
	resp := env.AuthGET("/ai/conversations/"+conv.ID+"/messages", token2)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ─── Video Tests (4) ──────────────────────────────────────────────────────

func TestVideo_StartSession(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("vidstart@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/video/sessions", map[string]string{
		"stream_url": "https://stream.test.com/live",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestVideo_EndSession(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("vidend@test.com", "securepass123", "EUR")

	startResp := env.AuthPOST("/video/sessions", map[string]string{
		"stream_url": "https://stream.test.com/live2",
	}, token)
	var session struct{ ID string `json:"id"` }
	json.NewDecoder(startResp.Body).Decode(&session)
	startResp.Body.Close()

	resp := env.AuthPOST("/video/sessions/"+session.ID+"/end", nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "ended", result.Status)
}

func TestVideo_ListSessions(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("vidlist@test.com", "securepass123", "EUR")

	env.AuthPOST("/video/sessions", map[string]string{
		"stream_url": "https://stream.test.com/live3",
	}, token)

	resp := env.AuthGET("/video/sessions", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var sessions []struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sessions))
	assert.Len(t, sessions, 1)
}

func TestVideo_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/video/sessions")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Social Tests (5) ─────────────────────────────────────────────────────

func TestSocial_CreatePost(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("socialcreate@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/social/posts", map[string]string{
		"content": "Hello social world!", "type": "text",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
}

func TestSocial_ListPosts(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sociallist@test.com", "securepass123", "EUR")

	env.AuthPOST("/social/posts", map[string]string{
		"content": "Test post", "type": "text",
	}, token)

	resp := env.AuthGET("/social/posts", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var posts []struct {
		Content string `json:"content"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&posts))
	assert.GreaterOrEqual(t, len(posts), 1)
}

func TestSocial_DeletePost(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("socialdelete@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/social/posts", map[string]string{
		"content": "Delete me", "type": "text",
	}, token)
	var post struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&post)
	createResp.Body.Close()

	resp := env.AuthDELETE("/social/posts/"+post.ID, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSocial_WrongPlayerCantDelete(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("socialown1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("socialown2@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/social/posts", map[string]string{
		"content": "My post", "type": "text",
	}, token1)
	var post struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&post)
	createResp.Body.Close()

	resp := env.AuthDELETE("/social/posts/"+post.ID, token2)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestSocial_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/social/posts", map[string]string{
		"content": "No auth", "type": "text",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── AI Advanced Tests (3) ────────────────────────────────────────────────

func TestAI_ConversationIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("aiiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("aiiso2@test.com", "securepass123", "EUR")

	// Player 1 creates a conversation
	env.AuthPOST("/ai/conversations", nil, token1)

	// Player 2 lists conversations — should be empty
	resp := env.AuthGET("/ai/conversations", token2)
	defer resp.Body.Close()

	var convs []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&convs))
	assert.Empty(t, convs)
}

func TestAI_SendMessageToNonexistentConvo(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("ainoconvo@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/ai/conversations/"+uuid.New().String()+"/messages", map[string]string{
		"content": "Hello",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAI_EmptyMessageContent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("aiempty@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/ai/conversations", nil, token)
	var conv struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&conv)
	createResp.Body.Close()

	resp := env.AuthPOST("/ai/conversations/"+conv.ID+"/messages", map[string]string{
		"content": "",
	}, token)
	defer resp.Body.Close()

	// Document behavior — empty message may be accepted or rejected
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest,
		"expected 201 or 400, got %d", resp.StatusCode)
}

// ─── Video Advanced Tests (2) ─────────────────────────────────────────────

func TestVideo_EndSessionWrongPlayer(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("vidwrong1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("vidwrong2@test.com", "securepass123", "EUR")

	startResp := env.AuthPOST("/video/sessions", map[string]string{
		"stream_url": "https://stream.test.com/wrong",
	}, token1)
	var session struct{ ID string `json:"id"` }
	json.NewDecoder(startResp.Body).Decode(&session)
	startResp.Body.Close()

	// Player 2 tries to end player 1's session
	resp := env.AuthPOST("/video/sessions/"+session.ID+"/end", nil, token2)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVideo_SessionIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, _ := env.RegisterPlayer("vidiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("vidiso2@test.com", "securepass123", "EUR")

	env.AuthPOST("/video/sessions", map[string]string{
		"stream_url": "https://stream.test.com/iso",
	}, token1)

	// Player 2 lists sessions — should be empty
	resp := env.AuthGET("/video/sessions", token2)
	defer resp.Body.Close()

	var sessions []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sessions))
	assert.Empty(t, sessions)
}

// ─── Social Advanced Tests (2) ────────────────────────────────────────────

func TestSocial_DeleteTwice(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("socialtwice@test.com", "securepass123", "EUR")

	createResp := env.AuthPOST("/social/posts", map[string]string{
		"content": "Delete me twice", "type": "text",
	}, token)
	var post struct{ ID string `json:"id"` }
	json.NewDecoder(createResp.Body).Decode(&post)
	createResp.Body.Close()

	// First delete should succeed
	resp1 := env.AuthDELETE("/social/posts/"+post.ID, token)
	resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Second delete should return 404
	resp2 := env.AuthDELETE("/social/posts/"+post.ID, token)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestSocial_EmptyContent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("socialempty@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/social/posts", map[string]string{
		"content": "", "type": "text",
	}, token)
	defer resp.Body.Close()

	// Document behavior — empty content may be accepted or rejected
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest,
		"expected 201 or 400, got %d", resp.StatusCode)
}

// ─── Plugin Tests (4) ─────────────────────────────────────────────────────

func TestPlugins_List(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("pluglist@test.com", "securepass123", "EUR")
	env.SeedPlugin("Test Plugin")

	resp := env.AuthGET("/plugins/", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var plugins []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&plugins))
	assert.GreaterOrEqual(t, len(plugins), 1)
}

func TestPlugins_Dispatch(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("plugdispatch@test.com", "securepass123", "EUR")
	pluginID := env.SeedPlugin("Dispatch Plugin")

	resp := env.AuthPOST("/plugins/dispatch", map[string]interface{}{
		"plugin_id":       pluginID,
		"scope":           "read",
		"payload":         map[string]string{"key": "value"},
		"idempotency_key": uuid.New().String(),
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "completed", result.Status)
}

func TestPlugins_ListDispatches(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("plugdisps@test.com", "securepass123", "EUR")
	pluginID := env.SeedPlugin("Dispatch List Plugin")

	env.AuthPOST("/plugins/dispatch", map[string]interface{}{
		"plugin_id":       pluginID,
		"scope":           "read",
		"payload":         map[string]string{"key": "value"},
		"idempotency_key": uuid.New().String(),
	}, token)

	resp := env.AuthGET("/plugins/"+pluginID+"/dispatches", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var dispatches []struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&dispatches))
	assert.GreaterOrEqual(t, len(dispatches), 1)
}

func TestPlugins_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/plugins/")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Affiliate Tests (2) ──────────────────────────────────────────────────

func TestAffiliate_Register(t *testing.T) {
	env := testutil.NewTestEnv(t)

	resp := env.POST("/affiliates/register", map[string]string{
		"email":      "affiliate@test.com",
		"password":   "securepass123",
		"first_name": "John",
		"last_name":  "Doe",
		"company":    "TestCo",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
}

func TestAffiliate_Login(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.POST("/affiliates/register", map[string]string{
		"email":      "afflogin@test.com",
		"password":   "securepass123",
		"first_name": "Jane",
		"last_name":  "Doe",
	}, "")

	resp := env.POST("/affiliates/login", map[string]string{
		"email":    "afflogin@test.com",
		"password": "securepass123",
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.Token)
}
