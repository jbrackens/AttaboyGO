//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Balance Tests (5) ─────────────────────────────────────────────────────

func TestBalance_NewPlayerZeros(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("newbal@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/wallet/balance", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var bal struct {
		Balance         int64  `json:"balance"`
		BonusBalance    int64  `json:"bonus_balance"`
		ReservedBalance int64  `json:"reserved_balance"`
		Currency        string `json:"currency"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bal))
	assert.Equal(t, int64(0), bal.Balance)
	assert.Equal(t, int64(0), bal.BonusBalance)
	assert.Equal(t, int64(0), bal.ReservedBalance)
}

func TestBalance_AfterDeposit(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("depbal@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)

	resp := env.AuthGET("/wallet/balance", token)
	defer resp.Body.Close()

	var bal struct {
		Balance int64 `json:"balance"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bal))
	assert.Equal(t, int64(10000), bal.Balance)
}

func TestBalance_CorrectCurrency(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("gbpbal@test.com", "securepass123", "GBP")

	resp := env.AuthGET("/wallet/balance", token)
	defer resp.Body.Close()

	var bal struct {
		Currency string `json:"currency"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bal))
	assert.Equal(t, "GBP", bal.Currency)
}

func TestBalance_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/wallet/balance")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestBalance_IsolatedBetweenPlayers(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, playerID1 := env.RegisterPlayer("iso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("iso2@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID1, 7500)

	resp1 := env.AuthGET("/wallet/balance", token1)
	defer resp1.Body.Close()
	var bal1 struct{ Balance int64 `json:"balance"` }
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&bal1))
	assert.Equal(t, int64(7500), bal1.Balance)

	resp2 := env.AuthGET("/wallet/balance", token2)
	defer resp2.Body.Close()
	var bal2 struct{ Balance int64 `json:"balance"` }
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&bal2))
	assert.Equal(t, int64(0), bal2.Balance)
}

// ─── Deposit via Ledger Tests (6) ──────────────────────────────────────────

func TestDeposit_CreditsBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("depcred@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	testutil.AssertBalance(t, env, playerID, 5000, 0, 0)
}

func TestDeposit_MultipleAdditive(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("depadd@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 3000)
	env.DirectDeposit(playerID, 2000)

	testutil.AssertBalance(t, env, playerID, 5000, 0, 0)
}

func TestDeposit_CreatesTransaction(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("deptx@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	count := testutil.CountTransactions(t, env, playerID)
	assert.Equal(t, 1, count)
}

func TestDeposit_SnapshotMatches(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("depsnap@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	var balAfter int64
	err := env.Pool.QueryRow(t.Context(),
		"SELECT balance_after FROM v2_transactions WHERE player_id = $1 ORDER BY created_at DESC LIMIT 1",
		playerID).Scan(&balAfter)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), balAfter)
}

func TestDeposit_OutboxEventCreated(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("depoutbox@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	count := testutil.CountOutboxEvents(t, env, playerID)
	assert.GreaterOrEqual(t, count, 1)
}

func TestDeposit_BonusCreditSeparate(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID := env.RegisterPlayer("depbonus@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)
	env.DirectBonusCredit(playerID, 2000)

	testutil.AssertBalance(t, env, playerID, 5000, 2000, 0)
}

// ─── Withdrawal Tests (6) ──────────────────────────────────────────────────

func TestWithdrawal_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("wdsuccess@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)

	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 5000}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Balance should decrease, reserved should increase
	testutil.AssertBalance(t, env, playerID, 5000, 0, 5000)
}

func TestWithdrawal_InsufficientBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("wdinsuf@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 1000)

	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 5000}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWithdrawal_ExactBalance(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("wdexact@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 3000)

	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 3000}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	testutil.AssertBalance(t, env, playerID, 0, 0, 3000)
}

func TestWithdrawal_CreatesTransaction(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("wdtx@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 2000}, token)
	defer resp.Body.Close()

	// 1 deposit + 1 withdrawal = at least 2 total
	count := testutil.CountTransactions(t, env, playerID)
	assert.GreaterOrEqual(t, count, 2)
}

func TestWithdrawal_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.POST("/payments/withdraw", map[string]int64{"amount": 1000}, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWithdrawal_ZeroAmount(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("wdzero@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 0}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Transaction History Tests (7) ─────────────────────────────────────────

func TestTransactions_EmptyList(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("txempty@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/wallet/transactions", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Transactions []json.RawMessage `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Empty(t, result.Transactions)
}

func TestTransactions_IncludesDeposit(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("txdep@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 5000)

	resp := env.AuthGET("/wallet/transactions", token)
	defer resp.Body.Close()

	var result struct {
		Transactions []struct {
			Type   string `json:"type"`
			Amount int64  `json:"amount"`
		} `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Transactions, 1)
	assert.Equal(t, "wallet_deposit", result.Transactions[0].Type)
	assert.Equal(t, int64(5000), result.Transactions[0].Amount)
}

func TestTransactions_DescOrder(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("txorder@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 1000)
	env.DirectDeposit(playerID, 2000)

	resp := env.AuthGET("/wallet/transactions", token)
	defer resp.Body.Close()

	var result struct {
		Transactions []struct {
			Amount int64 `json:"amount"`
		} `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Len(t, result.Transactions, 2)
	// Most recent (2000) should be first
	assert.Equal(t, int64(2000), result.Transactions[0].Amount)
	assert.Equal(t, int64(1000), result.Transactions[1].Amount)
}

func TestTransactions_DefaultLimit20(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("txlimit@test.com", "securepass123", "EUR")
	for i := 0; i < 25; i++ {
		env.DirectDeposit(playerID, int64(100*(i+1)))
	}

	resp := env.AuthGET("/wallet/transactions", token)
	defer resp.Body.Close()

	var result struct {
		Transactions []json.RawMessage `json:"transactions"`
		NextCursor   *string           `json:"next_cursor"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Transactions, 20)
	assert.NotNil(t, result.NextCursor)
}

func TestTransactions_CustomLimit(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("txcustom@test.com", "securepass123", "EUR")
	for i := 0; i < 10; i++ {
		env.DirectDeposit(playerID, int64(100*(i+1)))
	}

	resp := env.AuthGET("/wallet/transactions?limit=5", token)
	defer resp.Body.Close()

	var result struct {
		Transactions []json.RawMessage `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Transactions, 5)
}

func TestTransactions_CursorPagination(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("txcursor@test.com", "securepass123", "EUR")
	for i := 0; i < 5; i++ {
		env.DirectDeposit(playerID, int64(100*(i+1)))
	}

	// First page
	resp1 := env.AuthGET("/wallet/transactions?limit=3", token)
	defer resp1.Body.Close()

	var page1 struct {
		Transactions []json.RawMessage `json:"transactions"`
		NextCursor   *string           `json:"next_cursor"`
	}
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&page1))
	assert.Len(t, page1.Transactions, 3)
	assert.NotNil(t, page1.NextCursor)

	// Second page
	resp2 := env.AuthGET(fmt.Sprintf("/wallet/transactions?limit=3&cursor=%s", *page1.NextCursor), token)
	defer resp2.Body.Close()

	var page2 struct {
		Transactions []json.RawMessage `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&page2))
	assert.Len(t, page2.Transactions, 2)
}

func TestTransactions_PlayerIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	_, playerID1 := env.RegisterPlayer("txiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("txiso2@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID1, 5000)

	resp := env.AuthGET("/wallet/transactions", token2)
	defer resp.Body.Close()

	var result struct {
		Transactions []json.RawMessage `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Empty(t, result.Transactions)
}

// ─── Payment History Tests (6) ─────────────────────────────────────────────

func TestPaymentHistory_Empty(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("payempty@test.com", "securepass123", "EUR")

	resp := env.AuthGET("/payments/history", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPaymentHistory_AfterWithdrawal(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, playerID := env.RegisterPlayer("payhist@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID, 10000)

	env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 5000}, token)

	resp := env.AuthGET("/payments/history", token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var payments []struct {
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payments))
	assert.GreaterOrEqual(t, len(payments), 1)
}

func TestPaymentHistory_PlayerIsolation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token1, playerID1 := env.RegisterPlayer("payiso1@test.com", "securepass123", "EUR")
	token2, _ := env.RegisterPlayer("payiso2@test.com", "securepass123", "EUR")
	env.DirectDeposit(playerID1, 10000)
	env.AuthPOST("/payments/withdraw", map[string]int64{"amount": 5000}, token1)

	resp := env.AuthGET("/payments/history", token2)
	defer resp.Body.Close()

	var payments []json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payments))
	assert.Empty(t, payments)
}

func TestPaymentHistory_RequiresAuth(t *testing.T) {
	env := testutil.NewTestEnv(t)
	resp := env.GET("/payments/history")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPaymentHistory_DepositValidation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("paydepv@test.com", "securepass123", "EUR")

	// Zero amount should fail
	resp := env.AuthPOST("/payments/deposit", map[string]interface{}{
		"amount": 0, "currency": "EUR", "success_url": "http://x.com/ok", "cancel_url": "http://x.com/no",
	}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPaymentHistory_WithdrawValidation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	token, _ := env.RegisterPlayer("paywdv@test.com", "securepass123", "EUR")

	// Negative amount should fail
	resp := env.AuthPOST("/payments/withdraw", map[string]int64{"amount": -100}, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
