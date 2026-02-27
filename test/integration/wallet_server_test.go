//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- BetSolutions Tests ---

func TestBS_Balance(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.BSPost("/betsolutions/balance", provider.BetSolutionsRequest{
		Token:    "test-token",
		PlayerID: playerID.String(),
		Currency: "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int64(5000), result.Balance)
}

func TestBS_Bet(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	resp := env.BSPost("/betsolutions/bet", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-bet-1",
		Amount:        3000,
		Currency:      "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int64(7000), result.Balance)

	// Verify DB balance
	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(7000), bal)
}

func TestBS_Win(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Place bet first
	env.BSPost("/betsolutions/bet", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-bet-1",
		Amount:        3000,
		Currency:      "EUR",
	})

	// Credit win
	resp := env.BSPost("/betsolutions/win", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-win-1",
		Amount:        8000,
		Currency:      "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int64(15000), result.Balance) // 10000 - 3000 + 8000

	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(15000), bal)
}

func TestBS_Rollback_Found(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Place bet
	env.BSPost("/betsolutions/bet", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-bet-1",
		Amount:        3000,
		Currency:      "EUR",
	})

	// Rollback the bet
	resp := env.BSPost("/betsolutions/rollback", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-bet-1", // same as the bet
		Currency:      "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int64(10000), result.Balance) // fully restored

	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(10000), bal)
}

func TestBS_Rollback_NotFound(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Rollback a transaction that never existed
	resp := env.BSPost("/betsolutions/rollback", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		TransactionID: "tx-never-placed",
		Currency:      "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, int64(10000), result.Balance) // unchanged
}

func TestBS_InvalidSignature(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")

	resp := env.BSPostBadSig("/betsolutions/balance", provider.BetSolutionsRequest{
		Token:    "test-token",
		PlayerID: playerID.String(),
		Currency: "EUR",
	})
	defer resp.Body.Close()

	var result provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 401, result.StatusCode)
	assert.Equal(t, "invalid signature", result.Error)
}

func TestBS_IdempotentBet(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	req := provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-1",
		RoundID:       "round-1",
		TransactionID: "tx-idem-1",
		Amount:        2000,
		Currency:      "EUR",
	}

	// First bet
	resp1 := env.BSPost("/betsolutions/bet", req)
	defer resp1.Body.Close()
	var r1 provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&r1))
	assert.Equal(t, 200, r1.StatusCode)
	assert.Equal(t, int64(8000), r1.Balance)

	// Same bet again (idempotent)
	resp2 := env.BSPost("/betsolutions/bet", req)
	defer resp2.Body.Close()
	var r2 provider.BetSolutionsResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&r2))
	assert.Equal(t, 200, r2.StatusCode)
	assert.Equal(t, int64(8000), r2.Balance) // same balance, not deducted again

	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(8000), bal)
}

// --- Pragmatic Play Tests ---

func TestPP_Balance(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.PPPost(provider.PragmaticRequest{
		UserID:   playerID.String(),
		Action:   "balance",
		Currency: "EUR",
		Token:    "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, result.Error)
	assert.Equal(t, "50.00", result.Cash)
	assert.Equal(t, "0.00", result.Bonus)
}

func TestPP_Bet(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	resp := env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "bet",
		Amount:        "30.00",
		Currency:      "EUR",
		TransactionID: "pp-bet-1",
		RoundID:       "pp-round-1",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, result.Error)
	assert.Equal(t, "70.00", result.Cash) // 100.00 - 30.00
}

func TestPP_Result(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Bet first
	env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "bet",
		Amount:        "30.00",
		Currency:      "EUR",
		TransactionID: "pp-bet-1",
		RoundID:       "pp-round-1",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})

	// Win
	resp := env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "result",
		Amount:        "80.00",
		Currency:      "EUR",
		TransactionID: "pp-win-1",
		RoundID:       "pp-round-1",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, result.Error)
	assert.Equal(t, "150.00", result.Cash) // 100.00 - 30.00 + 80.00
}

func TestPP_Refund(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Bet first
	env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "bet",
		Amount:        "30.00",
		Currency:      "EUR",
		TransactionID: "pp-bet-1",
		RoundID:       "pp-round-1",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})

	// Refund
	resp := env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "refund",
		Currency:      "EUR",
		TransactionID: "pp-bet-1", // same as original
		RoundID:       "pp-round-1",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, result.Error)
	assert.Equal(t, "100.00", result.Cash) // fully restored
}

func TestPP_InvalidSignature(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")

	resp := env.PPPostBadSig(provider.PragmaticRequest{
		UserID:   playerID.String(),
		Action:   "balance",
		Currency: "EUR",
		Token:    "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 1, result.Error)
	assert.Equal(t, "invalid signature", result.Message)
}

func TestPP_DecimalAmounts(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	resp := env.PPPost(provider.PragmaticRequest{
		UserID:        playerID.String(),
		Action:        "bet",
		Amount:        "10.50",
		Currency:      "EUR",
		TransactionID: "pp-dec-1",
		RoundID:       "pp-round-dec",
		GameID:        "pp-game-1",
		Token:         "test-token",
	})
	defer resp.Body.Close()

	var result provider.PragmaticResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, result.Error)
	assert.Equal(t, "89.50", result.Cash) // 100.00 - 10.50

	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(8950), bal)
}

func TestWallet_BetWinSequence(t *testing.T) {
	env := testutil.NewWalletTestEnv(t)
	playerID := env.CreatePlayer("EUR")
	env.DirectDeposit(playerID, 10000)

	// Check balance via BS
	resp := env.BSPost("/betsolutions/balance", provider.BetSolutionsRequest{
		Token:    "test-token",
		PlayerID: playerID.String(),
		Currency: "EUR",
	})
	var balResp provider.BetSolutionsResponse
	json.NewDecoder(resp.Body).Decode(&balResp)
	resp.Body.Close()
	assert.Equal(t, int64(10000), balResp.Balance)

	// Bet 2000 via BS
	resp = env.BSPost("/betsolutions/bet", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-seq",
		RoundID:       "round-seq",
		TransactionID: "tx-seq-bet",
		Amount:        2000,
		Currency:      "EUR",
	})
	var betResp provider.BetSolutionsResponse
	json.NewDecoder(resp.Body).Decode(&betResp)
	resp.Body.Close()
	assert.Equal(t, int64(8000), betResp.Balance)

	// Win 5000 via BS
	resp = env.BSPost("/betsolutions/win", provider.BetSolutionsRequest{
		Token:         "test-token",
		PlayerID:      playerID.String(),
		GameID:        "game-seq",
		RoundID:       "round-seq",
		TransactionID: "tx-seq-win",
		Amount:        5000,
		Currency:      "EUR",
	})
	var winResp provider.BetSolutionsResponse
	json.NewDecoder(resp.Body).Decode(&winResp)
	resp.Body.Close()
	assert.Equal(t, int64(13000), winResp.Balance)

	// Final DB check
	bal, _ := env.GetBalance(playerID)
	assert.Equal(t, int64(13000), bal)
}
