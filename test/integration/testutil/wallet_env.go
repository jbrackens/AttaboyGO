//go:build integration

package testutil

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/attaboy/platform/internal/walletserver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TestBSSecret = "test-bs-secret"
	TestPPSecret = "test-pp-secret"
)

// WalletTestEnv holds resources for wallet server integration tests.
type WalletTestEnv struct {
	Server   *httptest.Server
	Pool     *pgxpool.Pool
	BSSecret string
	PPSecret string
	t        *testing.T
}

// NewWalletTestEnv creates a test environment for the wallet server.
func NewWalletTestEnv(t *testing.T) *WalletTestEnv {
	t.Helper()

	pool := getSharedPool(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	playerRepo := repository.NewPlayerRepository()
	txRepo := repository.NewTransactionRepository()
	outboxRepo := repository.NewOutboxRepository()
	eng := ledger.NewEngine(playerRepo, txRepo, outboxRepo)

	bsAdapter := provider.NewBetSolutionsAdapter(TestBSSecret, logger)
	ppAdapter := provider.NewPragmaticAdapter(TestPPSecret, logger)

	router := walletserver.NewRouter(pool, eng, txRepo, bsAdapter, ppAdapter, logger)
	server := httptest.NewServer(router)

	env := &WalletTestEnv{
		Server:   server,
		Pool:     pool,
		BSSecret: TestBSSecret,
		PPSecret: TestPPSecret,
		t:        t,
	}

	t.Cleanup(func() {
		server.Close()
		env.CleanAll()
	})

	env.CleanAll()
	return env
}

// CleanAll truncates all tables (delegates to shared cleanup logic).
func (env *WalletTestEnv) CleanAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tables := []string{
		"event_outbox",
		"v2_transactions",
		"player_profiles",
		"auth_users",
		"v2_players",
	}
	for _, table := range tables {
		_, _ = env.Pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE")
	}
}

// CreatePlayer inserts a player directly into v2_players and returns the ID.
func (env *WalletTestEnv) CreatePlayer(currency string) uuid.UUID {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	playerID := uuid.New()
	_, err := env.Pool.Exec(ctx, `
		INSERT INTO v2_players (id, currency, balance, bonus_balance, reserved_balance)
		VALUES ($1, $2, 0, 0, 0)`, playerID, currency)
	if err != nil {
		env.t.Fatalf("CreatePlayer: %v", err)
	}
	return playerID
}

// DirectDeposit credits a player's balance directly.
func (env *WalletTestEnv) DirectDeposit(playerID uuid.UUID, amountCents int64) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	extTxID := fmt.Sprintf("test_dep_%s", uuid.New().String()[:8])

	tx, err := env.Pool.Begin(ctx)
	if err != nil {
		env.t.Fatalf("DirectDeposit: begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SELECT id FROM v2_players WHERE id = $1 FOR UPDATE", playerID)
	if err != nil {
		env.t.Fatalf("DirectDeposit: lock: %v", err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE v2_players SET balance = balance + $2, updated_at = now() WHERE id = $1",
		playerID, amountCents)
	if err != nil {
		env.t.Fatalf("DirectDeposit: update balance: %v", err)
	}

	var balAfter, bonusAfter, reservedAfter int64
	err = tx.QueryRow(ctx,
		"SELECT balance, bonus_balance, reserved_balance FROM v2_players WHERE id = $1",
		playerID).Scan(&balAfter, &bonusAfter, &reservedAfter)
	if err != nil {
		env.t.Fatalf("DirectDeposit: read balance: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO v2_transactions (player_id, type, amount, balance_after, bonus_balance_after,
			reserved_balance_after, external_transaction_id, manufacturer_id, sub_transaction_id, metadata)
		VALUES ($1, 'wallet_deposit', $2, $3, $4, $5, $6, 'test', '1', '{}')`,
		playerID, amountCents, balAfter, bonusAfter, reservedAfter, extTxID)
	if err != nil {
		env.t.Fatalf("DirectDeposit: insert tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO event_outbox ("eventId", "aggregateType", "aggregateId", "eventType",
			"partitionKey", headers, payload, "occurredAt")
		VALUES ($1, 'wallet', $2, 'pam.wallet.transaction.posted', $2, '{}', '{}', now())`,
		uuid.New().String(), playerID.String())
	if err != nil {
		env.t.Fatalf("DirectDeposit: insert outbox: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		env.t.Fatalf("DirectDeposit: commit: %v", err)
	}
}

// GetBalance reads the current balance and bonus balance for a player.
func (env *WalletTestEnv) GetBalance(playerID uuid.UUID) (balance, bonusBalance int64) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := env.Pool.QueryRow(ctx,
		"SELECT balance, bonus_balance FROM v2_players WHERE id = $1",
		playerID).Scan(&balance, &bonusBalance)
	if err != nil {
		env.t.Fatalf("GetBalance: %v", err)
	}
	return
}

// BSPost sends a signed BetSolutions request and returns the response.
func (env *WalletTestEnv) BSPost(path string, req provider.BetSolutionsRequest) *http.Response {
	env.t.Helper()

	// Marshal without Hash to compute signature
	req.Hash = ""
	bodyNoHash, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("BSPost: marshal: %v", err)
	}

	// Compute HMAC over body without Hash field
	hash := computeHMAC(bodyNoHash, env.BSSecret)

	// Set Hash and re-marshal
	req.Hash = hash
	body, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("BSPost: re-marshal: %v", err)
	}

	resp, err := http.Post(env.Server.URL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		env.t.Fatalf("BSPost: %v", err)
	}
	return resp
}

// BSPostBadSig sends a BetSolutions request with an invalid signature.
func (env *WalletTestEnv) BSPostBadSig(path string, req provider.BetSolutionsRequest) *http.Response {
	env.t.Helper()

	req.Hash = "bad-signature"
	body, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("BSPostBadSig: marshal: %v", err)
	}

	resp, err := http.Post(env.Server.URL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		env.t.Fatalf("BSPostBadSig: %v", err)
	}
	return resp
}

// PPPost sends a signed Pragmatic request and returns the response.
func (env *WalletTestEnv) PPPost(req provider.PragmaticRequest) *http.Response {
	env.t.Helper()

	// Marshal without hash to compute signature
	req.ProvidedHash = ""
	bodyNoHash, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("PPPost: marshal: %v", err)
	}

	// Compute HMAC over body without hash field
	hash := computePPHMAC(bodyNoHash, env.PPSecret)

	// Set hash and re-marshal
	req.ProvidedHash = hash
	body, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("PPPost: re-marshal: %v", err)
	}

	resp, err := http.Post(env.Server.URL+"/pragmatic/", "application/json", bytes.NewReader(body))
	if err != nil {
		env.t.Fatalf("PPPost: %v", err)
	}
	return resp
}

// PPPostBadSig sends a Pragmatic request with an invalid signature.
func (env *WalletTestEnv) PPPostBadSig(req provider.PragmaticRequest) *http.Response {
	env.t.Helper()

	req.ProvidedHash = "bad-signature"
	body, err := json.Marshal(req)
	if err != nil {
		env.t.Fatalf("PPPostBadSig: marshal: %v", err)
	}

	resp, err := http.Post(env.Server.URL+"/pragmatic/", "application/json", bytes.NewReader(body))
	if err != nil {
		env.t.Fatalf("PPPostBadSig: %v", err)
	}
	return resp
}

// computeHMAC computes HMAC-SHA256 for BetSolutions (strips "Hash" field).
func computeHMAC(body []byte, secret string) string {
	// Strip the Hash field (same logic as the adapter)
	stripped := stripField(body, "Hash")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(stripped)
	return hex.EncodeToString(mac.Sum(nil))
}

// computePPHMAC computes HMAC-SHA256 for Pragmatic (strips "hash" field).
func computePPHMAC(body []byte, secret string) string {
	stripped := stripField(body, "hash")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(stripped)
	return hex.EncodeToString(mac.Sum(nil))
}

// stripField removes a top-level JSON field and re-marshals.
func stripField(body []byte, field string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	delete(m, field)
	stripped, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return stripped
}
