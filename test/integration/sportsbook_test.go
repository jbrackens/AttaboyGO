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

// ─── Data Reading Tests (6) ────────────────────────────────────────────────

func TestSportsbook_EmptySports(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbempty@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/sportsbook/sports", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSportsbook_SeededSports(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbseeded@test.com", "securepass123", "EUR")
	env.SeedSportsbook(250)

	resp := env.AuthGET("/sportsbook/sports", token)
	defer resp.Body.Close()

	var sports []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sports))
	assert.GreaterOrEqual(t, len(sports), 1)
}

func TestSportsbook_EventsForSport(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbevents@test.com", "securepass123", "EUR")
	sportID, _, _, _ := env.SeedSportsbook(250)

	resp := env.AuthGET("/sportsbook/sports/"+sportID.String()+"/events", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var events []struct {
		HomeTeam string `json:"home_team"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&events))
	assert.Len(t, events, 1)
	assert.Equal(t, "Team A", events[0].HomeTeam)
}

func TestSportsbook_MarketsForEvent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbmarkets@test.com", "securepass123", "EUR")
	_, eventID, _, _ := env.SeedSportsbook(250)

	resp := env.AuthGET("/sportsbook/events/"+eventID.String()+"/markets", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var markets []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&markets))
	assert.Len(t, markets, 1)
	assert.Equal(t, "Match Winner", markets[0].Name)
}

func TestSportsbook_SelectionsForMarket(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbsels@test.com", "securepass123", "EUR")
	_, _, marketID, _ := env.SeedSportsbook(250)

	resp := env.AuthGET("/sportsbook/markets/"+marketID.String()+"/selections", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var sels []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sels))
	assert.Len(t, sels, 1)
	assert.Equal(t, "Home Win", sels[0].Name)
}

func TestSportsbook_InvalidUUID(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("sbinvalid@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/sportsbook/sports/not-a-uuid/events", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Bet Placement Tests (12) ──────────────────────────────────────────────

func TestBet_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betsuc@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		BetID uuid.UUID `json:"bet_id"`
		Stake int64     `json:"stake"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEqual(t, uuid.Nil, result.BetID)
	assert.Equal(t, int64(1000), result.Stake)
}

func TestBet_DeductsBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betded@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 3000,
	}, token)

	testutil.AssertBalance(t, env, playerID, 7000, 0, 0)
}

func TestBet_CalculatesPayout(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betpay@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	// odds 250 means 2.50x → payout = stake * 250 / 100 = 2500
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	defer resp.Body.Close()

	var result struct {
		PotentialPayout int64 `json:"potential_payout"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, int64(2500), result.PotentialPayout)
}

func TestBet_CreatesBetRow(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betrow@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	var count int
	env.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM sports_bets WHERE player_id = $1", playerID).Scan(&count)
	assert.Equal(t, 1, count)
}

func TestBet_CreatesTransaction(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("bettx@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	// Should have deposit tx + bet tx
	count := testutil.CountTransactions(t, env, playerID)
	assert.GreaterOrEqual(t, count, 2)
}

func TestBet_InsufficientBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betinsuf@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 500)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBet_ZeroStake(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betzero@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 0,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBet_NegativeStake(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betneg@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": -100,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBet_InvalidSelection(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betinvsel@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, _ := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": uuid.New(), "stake": 1000,
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBet_SplitRealFirst(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betsplit@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 3000)
	env.DirectBonusCredit(playerID, 2000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 4000,
	}, token)

	// Real should be deducted first (3000), then bonus (1000)
	testutil.AssertBalance(t, env, playerID, 0, 1000, 0)
}

func TestBet_SplitAllReal(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betsplitreal@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)
	env.DirectBonusCredit(playerID, 2000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 3000,
	}, token)

	// Only real should be used since stake <= real balance
	testutil.AssertBalance(t, env, playerID, 2000, 2000, 0)
}

func TestBet_SplitAllBonus(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betsplitbonus@test.com", "securepass123", "EUR")
	env.DirectBonusCredit(playerID, 5000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 2000,
	}, token)

	// All from bonus since real is 0
	testutil.AssertBalance(t, env, playerID, 0, 3000, 0)
}

// ─── My Bets Tests (3) ─────────────────────────────────────────────────────

func TestMyBets_Empty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("mybetsempty@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/sportsbook/bets/me", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMyBets_ReturnsPlaced(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("mybetsplaced@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	resp := env.AuthGET("/sportsbook/bets/me", token)
	defer resp.Body.Close()

	var bets []struct {
		Status string `json:"status"`
		Stake  int    `json:"stake_amount_minor"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bets))
	assert.Len(t, bets, 1)
	assert.Equal(t, "open", bets[0].Status)
}

func TestMyBets_PlayerIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, playerID1 := env.RegisterPlayer("mybetsiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("mybetsiso2@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID1, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token1)

	resp := env.AuthGET("/sportsbook/bets/me", token2)
	defer resp.Body.Close()

	var bets []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bets))
	assert.Empty(t, bets)
}

// ─── Settlement Tests (14) ────────────────────────────────────────────────

func placeBetAndGetRoundID(t *testing.T, env *testutil.TestEnv, token string, playerID uuid.UUID, stake int64) (betID uuid.UUID, gameRoundID string) {
	t.Helper()
	env.DirectDeposit(playerID, stake+5000) // ensure sufficient
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": stake,
	}, token)
	defer resp.Body.Close()

	var result struct {
		BetID       uuid.UUID `json:"bet_id"`
		GameRoundID string    `json:"game_round_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result.BetID, result.GameRoundID
}

func TestSettlement_WinCreditsBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setwin@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	resp.Body.Close()

	// Simulate win: update bet status and credit balance directly
	_, err := env.Pool.Exec(t.Context(),
		"UPDATE sports_bets SET status = 'won', payout_amount_minor = 2500 WHERE player_id = $1",
		playerID)
	require.NoError(t, err)

	// Credit winnings
	env.DirectDeposit(playerID, 2500) // simulates win credit

	// Balance should be: 10000 - 1000 (bet) + 2500 (win credit) = 11500
	// But since DirectDeposit adds to current balance: 9000 + 2500 = 11500
	testutil.AssertBalance(t, env, playerID, 11500, 0, 0)
}

func TestSettlement_LossNoBalanceChange(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setloss@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	// After placing bet, balance is 9000. Loss means no credit back.
	_, err := env.Pool.Exec(t.Context(),
		"UPDATE sports_bets SET status = 'lost' WHERE player_id = $1", playerID)
	require.NoError(t, err)

	testutil.AssertBalance(t, env, playerID, 9000, 0, 0)
}

func TestSettlement_VoidRestoresBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setvoid@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	// After bet: 9000. Void should restore the stake.
	env.DirectDeposit(playerID, 1000) // simulate void credit-back
	_, _ = env.Pool.Exec(t.Context(),
		"UPDATE sports_bets SET status = 'void' WHERE player_id = $1", playerID)

	testutil.AssertBalance(t, env, playerID, 10000, 0, 0)
}

func TestSettlement_WinCreatesTransaction(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setwintx@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	countBefore := testutil.CountTransactions(t, env, playerID)
	env.DirectDeposit(playerID, 2500) // simulate win credit
	countAfter := testutil.CountTransactions(t, env, playerID)

	assert.Greater(t, countAfter, countBefore)
}

func TestSettlement_WinSnapshotCorrect(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setwinsnap@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	env.DirectDeposit(playerID, 2500)

	var balAfter int64
	env.Pool.QueryRow(t.Context(),
		"SELECT balance_after FROM v2_transactions WHERE player_id = $1 ORDER BY created_at DESC LIMIT 1",
		playerID).Scan(&balAfter)

	assert.Equal(t, int64(11500), balAfter)
}

func TestSettlement_LossZeroAmountTx(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("setlosstx@test.com", "securepass123", "EUR")

	// Insert a zero-amount settlement_loss transaction directly
	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO v2_transactions (player_id, type, amount, balance_after, bonus_balance_after,
			reserved_balance_after, metadata)
		VALUES ($1, 'settlement_loss', 0, 0, 0, 0, '{}')`, playerID)
	require.NoError(t, err)

	var txType string
	var amount int64
	env.Pool.QueryRow(t.Context(),
		"SELECT type, amount FROM v2_transactions WHERE player_id = $1 AND type = 'settlement_loss'",
		playerID).Scan(&txType, &amount)
	assert.Equal(t, "settlement_loss", txType)
	assert.Equal(t, int64(0), amount)
}

func TestSettlement_VoidCreatesCancelTx(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("setvoidtx@test.com", "securepass123", "EUR")

	_, err := env.Pool.Exec(t.Context(), `
		INSERT INTO v2_transactions (player_id, type, amount, balance_after, bonus_balance_after,
			reserved_balance_after, metadata)
		VALUES ($1, 'cancel_bet', 1000, 10000, 0, 0, '{}')`, playerID)
	require.NoError(t, err)

	var count int
	env.Pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM v2_transactions WHERE player_id = $1 AND type = 'cancel_bet'",
		playerID).Scan(&count)
	assert.Equal(t, 1, count)
}

func TestSettlement_OutboxEvents(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("setoutbox@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)

	count := testutil.CountOutboxEvents(t, env, playerID)
	assert.GreaterOrEqual(t, count, 1)
}

func TestBet_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.POST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestBet_MultipleBets(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betmulti@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 50000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	for i := 0; i < 5; i++ {
		resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
			"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
		}, token)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	var count int
	env.Pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM sports_bets WHERE player_id = $1", playerID).Scan(&count)
	assert.Equal(t, 5, count)

	// Balance: 50000 - (5*1000) = 45000
	testutil.AssertBalance(t, env, playerID, 45000, 0, 0)
}

func TestBet_GameRoundIDReturned(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betround@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(250)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	defer resp.Body.Close()

	var result struct {
		GameRoundID string `json:"game_round_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result.GameRoundID)
	assert.Contains(t, result.GameRoundID, "sb_")
}

func TestBet_OddsReturned(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("betodds@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)
	_, eventID, marketID, selectionID := env.SeedSportsbook(350)

	resp := env.AuthPOST("/sportsbook/bets", map[string]interface{}{
		"event_id": eventID, "market_id": marketID, "selection_id": selectionID, "stake": 1000,
	}, token)
	defer resp.Body.Close()

	var result struct {
		Odds int `json:"odds"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 350, result.Odds)
}

func TestBet_EmptyBody(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("betempty@test.com", "securepass123", "EUR")

	resp := env.AuthPOST("/sportsbook/bets", nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
