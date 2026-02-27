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
