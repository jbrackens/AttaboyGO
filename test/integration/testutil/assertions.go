//go:build integration

package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// DecodeJSON reads and decodes a JSON response body into dst.
func DecodeJSON(t *testing.T, resp *http.Response, dst interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
}

// AssertStatus checks that the response has the expected HTTP status code.
func AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("expected status %d, got %d", expected, resp.StatusCode)
	}
}

// AssertErrorCode checks that the response body contains the expected error code.
func AssertErrorCode(t *testing.T, resp *http.Response, expectedCode string) {
	t.Helper()
	var errResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	DecodeJSON(t, resp, &errResp)
	if errResp.Code != expectedCode {
		t.Errorf("expected error code %q, got %q (message: %s)", expectedCode, errResp.Code, errResp.Message)
	}
}

// AssertBalance queries the v2_players table and asserts the player's balances.
func AssertBalance(t *testing.T, env *TestEnv, playerID uuid.UUID, balance, bonus, reserved int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var bal, bon, res int64
	err := env.Pool.QueryRow(ctx,
		"SELECT balance, bonus_balance, reserved_balance FROM v2_players WHERE id = $1",
		playerID).Scan(&bal, &bon, &res)
	if err != nil {
		t.Fatalf("AssertBalance: query: %v", err)
	}
	if bal != balance {
		t.Errorf("balance: expected %d, got %d", balance, bal)
	}
	if bon != bonus {
		t.Errorf("bonus_balance: expected %d, got %d", bonus, bon)
	}
	if res != reserved {
		t.Errorf("reserved_balance: expected %d, got %d", reserved, res)
	}
}

// CountTransactions returns the number of transactions for a player.
func CountTransactions(t *testing.T, env *TestEnv, playerID uuid.UUID) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := env.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM v2_transactions WHERE player_id = $1", playerID).Scan(&count)
	if err != nil {
		t.Fatalf("CountTransactions: %v", err)
	}
	return count
}

// CountOutboxEvents returns the number of outbox events for a player.
func CountOutboxEvents(t *testing.T, env *TestEnv, playerID uuid.UUID) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := env.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_outbox WHERE "aggregateId" = $1`, playerID.String()).Scan(&count)
	if err != nil {
		t.Fatalf("CountOutboxEvents: %v", err)
	}
	return count
}
