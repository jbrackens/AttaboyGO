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
	"net/http"
	"strconv"
	"time"

	"github.com/attaboy/platform/internal/auth"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegisterPlayer creates a new player and returns the auth token and player ID.
func (env *TestEnv) RegisterPlayer(email, password, currency string) (token string, playerID uuid.UUID) {
	env.t.Helper()
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	if currency != "" {
		body["currency"] = currency
	}

	resp := env.POST("/auth/register", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		env.t.Fatalf("RegisterPlayer: expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		Token    string    `json:"token"`
		PlayerID uuid.UUID `json:"player_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		env.t.Fatalf("RegisterPlayer: decode: %v", err)
	}
	return result.Token, result.PlayerID
}

// LoginPlayer authenticates an existing player and returns the auth token.
func (env *TestEnv) LoginPlayer(email, password string) string {
	env.t.Helper()
	resp := env.POST("/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		env.t.Fatalf("LoginPlayer: expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		env.t.Fatalf("LoginPlayer: decode: %v", err)
	}
	return result.Token
}

// GET performs an unauthenticated GET request.
func (env *TestEnv) GET(path string) *http.Response {
	env.t.Helper()
	resp, err := http.Get(env.Server.URL + path)
	if err != nil {
		env.t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// POST performs a POST request with optional auth token.
func (env *TestEnv) POST(path string, body interface{}, token string) *http.Response {
	env.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			env.t.Fatalf("POST %s: encode: %v", path, err)
		}
	}
	req, err := http.NewRequest("POST", env.Server.URL+path, &buf)
	if err != nil {
		env.t.Fatalf("POST %s: new request: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// AuthGET performs an authenticated GET request.
func (env *TestEnv) AuthGET(path, token string) *http.Response {
	env.t.Helper()
	req, err := http.NewRequest("GET", env.Server.URL+path, nil)
	if err != nil {
		env.t.Fatalf("AuthGET %s: new request: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("AuthGET %s: %v", path, err)
	}
	return resp
}

// AuthPOST performs an authenticated POST request.
func (env *TestEnv) AuthPOST(path string, body interface{}, token string) *http.Response {
	env.t.Helper()
	return env.POST(path, body, token)
}

// AuthPATCH performs an authenticated PATCH request.
func (env *TestEnv) AuthPATCH(path string, body interface{}, token string) *http.Response {
	env.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			env.t.Fatalf("PATCH %s: encode: %v", path, err)
		}
	}
	req, err := http.NewRequest("PATCH", env.Server.URL+path, &buf)
	if err != nil {
		env.t.Fatalf("PATCH %s: new request: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("PATCH %s: %v", path, err)
	}
	return resp
}

// AuthDELETE performs an authenticated DELETE request.
func (env *TestEnv) AuthDELETE(path, token string) *http.Response {
	env.t.Helper()
	req, err := http.NewRequest("DELETE", env.Server.URL+path, nil)
	if err != nil {
		env.t.Fatalf("DELETE %s: new request: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

// OPTIONS performs an OPTIONS request.
func (env *TestEnv) OPTIONS(path string) *http.Response {
	env.t.Helper()
	req, err := http.NewRequest("OPTIONS", env.Server.URL+path, nil)
	if err != nil {
		env.t.Fatalf("OPTIONS %s: new request: %v", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("OPTIONS %s: %v", path, err)
	}
	return resp
}

// DirectDeposit credits a player's balance directly via the ledger engine (bypasses Stripe).
func (env *TestEnv) DirectDeposit(playerID uuid.UUID, amountCents int64) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	extTxID := fmt.Sprintf("test_dep_%s", uuid.New().String()[:8])

	tx, err := env.Pool.Begin(ctx)
	if err != nil {
		env.t.Fatalf("DirectDeposit: begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	// Lock player
	_, err = tx.Exec(ctx, "SELECT id FROM v2_players WHERE id = $1 FOR UPDATE", playerID)
	if err != nil {
		env.t.Fatalf("DirectDeposit: lock: %v", err)
	}

	// Update balance
	_, err = tx.Exec(ctx,
		"UPDATE v2_players SET balance = balance + $2, updated_at = now() WHERE id = $1",
		playerID, amountCents)
	if err != nil {
		env.t.Fatalf("DirectDeposit: update balance: %v", err)
	}

	// Get updated balances for snapshot
	var balAfter, bonusAfter, reservedAfter int64
	err = tx.QueryRow(ctx,
		"SELECT balance, bonus_balance, reserved_balance FROM v2_players WHERE id = $1",
		playerID).Scan(&balAfter, &bonusAfter, &reservedAfter)
	if err != nil {
		env.t.Fatalf("DirectDeposit: read balance: %v", err)
	}

	// Insert transaction
	_, err = tx.Exec(ctx, `
		INSERT INTO v2_transactions (player_id, type, amount, balance_after, bonus_balance_after,
			reserved_balance_after, external_transaction_id, manufacturer_id, sub_transaction_id, metadata)
		VALUES ($1, 'wallet_deposit', $2, $3, $4, $5, $6, 'test', '1', '{}')`,
		playerID, amountCents, balAfter, bonusAfter, reservedAfter, extTxID)
	if err != nil {
		env.t.Fatalf("DirectDeposit: insert tx: %v", err)
	}

	// Insert outbox event
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

// DirectBonusCredit credits a player's bonus_balance directly.
func (env *TestEnv) DirectBonusCredit(playerID uuid.UUID, amountCents int64) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	extTxID := fmt.Sprintf("test_bonus_%s", uuid.New().String()[:8])

	tx, err := env.Pool.Begin(ctx)
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SELECT id FROM v2_players WHERE id = $1 FOR UPDATE", playerID)
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: lock: %v", err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE v2_players SET bonus_balance = bonus_balance + $2, updated_at = now() WHERE id = $1",
		playerID, amountCents)
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: update: %v", err)
	}

	var balAfter, bonusAfter, reservedAfter int64
	err = tx.QueryRow(ctx,
		"SELECT balance, bonus_balance, reserved_balance FROM v2_players WHERE id = $1",
		playerID).Scan(&balAfter, &bonusAfter, &reservedAfter)
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: read: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO v2_transactions (player_id, type, amount, balance_after, bonus_balance_after,
			reserved_balance_after, external_transaction_id, manufacturer_id, sub_transaction_id, metadata)
		VALUES ($1, 'bonus_credit', $2, $3, $4, $5, $6, 'test', '1', '{}')`,
		playerID, amountCents, balAfter, bonusAfter, reservedAfter, extTxID)
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: insert tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO event_outbox ("eventId", "aggregateType", "aggregateId", "eventType",
			"partitionKey", headers, payload, "occurredAt")
		VALUES ($1, 'wallet', $2, 'pam.wallet.transaction.posted', $2, '{}', '{}', now())`,
		uuid.New().String(), playerID.String())
	if err != nil {
		env.t.Fatalf("DirectBonusCredit: insert outbox: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		env.t.Fatalf("DirectBonusCredit: commit: %v", err)
	}
}

// AdminToken generates a JWT for an admin user with the given role.
func (env *TestEnv) AdminToken(role string) string {
	env.t.Helper()
	token, err := env.JWTMgr.GenerateToken(auth.RealmAdmin, uuid.New(), "admin@test.com", role, "")
	if err != nil {
		env.t.Fatalf("AdminToken: %v", err)
	}
	return token
}

// SeedSportsbook inserts a sport, event, market, and selection, returning their IDs.
func (env *TestEnv) SeedSportsbook(oddsDecimal int) (sportID, eventID, marketID, selectionID uuid.UUID) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sportID = uuid.New()
	eventID = uuid.New()
	marketID = uuid.New()
	selectionID = uuid.New()

	_, err := env.Pool.Exec(ctx, `
		INSERT INTO sports (id, key, name, icon, sort_order, active)
		VALUES ($1, $2, 'Test Football', '⚽', 1, true)`,
		sportID, fmt.Sprintf("football_%s", sportID.String()[:8]))
	if err != nil {
		env.t.Fatalf("SeedSportsbook: insert sport: %v", err)
	}

	_, err = env.Pool.Exec(ctx, `
		INSERT INTO sports_events (id, sport_id, league, home_team, away_team, start_time, status)
		VALUES ($1, $2, 'Premier League', 'Team A', 'Team B', $3, 'upcoming')`,
		eventID, sportID, time.Now().Add(24*time.Hour))
	if err != nil {
		env.t.Fatalf("SeedSportsbook: insert event: %v", err)
	}

	_, err = env.Pool.Exec(ctx, `
		INSERT INTO sports_markets (id, event_id, name, type, status, specifiers, sort_order)
		VALUES ($1, $2, 'Match Winner', '1x2', 'open', '', 1)`,
		marketID, eventID)
	if err != nil {
		env.t.Fatalf("SeedSportsbook: insert market: %v", err)
	}

	_, err = env.Pool.Exec(ctx, `
		INSERT INTO sports_selections (id, market_id, name, odds_decimal, odds_fractional, odds_american, status, sort_order)
		VALUES ($1, $2, 'Home Win', $3, '2/1', '+200', 'active', 1)`,
		selectionID, marketID, oddsDecimal)
	if err != nil {
		env.t.Fatalf("SeedSportsbook: insert selection: %v", err)
	}

	return sportID, eventID, marketID, selectionID
}

// SeedQuest inserts a quest and returns its ID.
func (env *TestEnv) SeedQuest(name string, targetProgress, rewardAmount int) uuid.UUID {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var questID uuid.UUID
	err := env.Pool.QueryRow(ctx, `
		INSERT INTO quests (name, description, type, target_progress, reward_amount, reward_currency, min_score, active, sort_order)
		VALUES ($1, 'Test quest', 'standard', $2, $3, 'EUR', 0, true, 1) RETURNING id`,
		name, targetProgress, rewardAmount).Scan(&questID)
	if err != nil {
		env.t.Fatalf("SeedQuest: %v", err)
	}
	return questID
}

// SeedPredictionMarket inserts a prediction market and returns its ID.
func (env *TestEnv) SeedPredictionMarket(title string) uuid.UUID {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Need an admin user for created_by FK
	adminID := uuid.New()
	_, _ = env.Pool.Exec(ctx, `
		INSERT INTO admin_users (id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'hash', 'Test Admin', 'superadmin')
		ON CONFLICT (email) DO NOTHING`,
		adminID, fmt.Sprintf("admin_%s@test.com", adminID.String()[:8]))

	// Get the admin ID (may be the existing one)
	env.Pool.QueryRow(ctx, "SELECT id FROM admin_users LIMIT 1").Scan(&adminID)

	var marketID uuid.UUID
	err := env.Pool.QueryRow(ctx, `
		INSERT INTO prediction_markets (title, description, category, status, created_by)
		VALUES ($1, 'Test prediction', 'general', 'open', $2) RETURNING id`,
		title, adminID).Scan(&marketID)
	if err != nil {
		env.t.Fatalf("SeedPredictionMarket: %v", err)
	}
	return marketID
}

// RegisterAffiliate creates a new affiliate and returns the auth token and affiliate ID.
func (env *TestEnv) RegisterAffiliate(email, password, firstName, lastName string) (token string, affiliateID uuid.UUID) {
	env.t.Helper()
	body := map[string]string{
		"email":      email,
		"password":   password,
		"first_name": firstName,
		"last_name":  lastName,
	}

	resp := env.POST("/affiliates/register", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		env.t.Fatalf("RegisterAffiliate: expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		Token       string    `json:"token"`
		AffiliateID uuid.UUID `json:"player_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		env.t.Fatalf("RegisterAffiliate: decode: %v", err)
	}
	return result.Token, result.AffiliateID
}

// LoginAffiliate authenticates an existing affiliate and returns the auth token.
func (env *TestEnv) LoginAffiliate(email, password string) string {
	env.t.Helper()
	resp := env.POST("/affiliates/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		env.t.Fatalf("LoginAffiliate: expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		env.t.Fatalf("LoginAffiliate: decode: %v", err)
	}
	return result.Token
}

// SeedAffiliateLink inserts an affiliate link for the given affiliate and returns the btag.
func (env *TestEnv) SeedAffiliateLink(affiliateID uuid.UUID, btag string) {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := env.Pool.Exec(ctx, `
		INSERT INTO affiliate_links (affiliate_id, btag, target_url, active)
		VALUES ($1, $2, 'https://example.com', true)`,
		affiliateID, btag)
	if err != nil {
		env.t.Fatalf("SeedAffiliateLink: %v", err)
	}
}

// RawPOST performs a POST request with raw bytes and custom headers.
func (env *TestEnv) RawPOST(path string, body []byte, headers map[string]string) *http.Response {
	env.t.Helper()
	req, err := http.NewRequest("POST", env.Server.URL+path, bytes.NewReader(body))
	if err != nil {
		env.t.Fatalf("RawPOST %s: new request: %v", path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("RawPOST %s: %v", path, err)
	}
	return resp
}

// GETWithHeaders performs a GET request with custom headers.
func (env *TestEnv) GETWithHeaders(path string, headers map[string]string) *http.Response {
	env.t.Helper()
	req, err := http.NewRequest("GET", env.Server.URL+path, nil)
	if err != nil {
		env.t.Fatalf("GETWithHeaders %s: new request: %v", path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		env.t.Fatalf("GETWithHeaders %s: %v", path, err)
	}
	return resp
}

// SeedPlugin inserts a plugin and returns its plugin_id.
func (env *TestEnv) SeedPlugin(name string) string {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pluginID := fmt.Sprintf("plugin_%s", uuid.New().String()[:8])
	_, err := env.Pool.Exec(ctx, `
		INSERT INTO plugins (plugin_id, name, description, domain, scopes, risk_tier, active)
		VALUES ($1, $2, 'Test plugin', 'test', '["read"]', 'standard', true)`,
		pluginID, name)
	if err != nil {
		env.t.Fatalf("SeedPlugin: %v", err)
	}
	return pluginID
}

// RegisterAdmin inserts an admin user directly into the DB and returns a JWT.
func (env *TestEnv) RegisterAdmin(email, password, role string) string {
	env.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		env.t.Fatalf("RegisterAdmin: hash: %v", err)
	}

	_, err = env.Pool.Exec(ctx, `
		INSERT INTO admin_users (id, email, password_hash, display_name, role)
		VALUES ($1, $2, $3, 'Test Admin', $4)`,
		adminID, email, string(hash), role)
	if err != nil {
		env.t.Fatalf("RegisterAdmin: insert: %v", err)
	}

	token, err := env.JWTMgr.GenerateToken(auth.RealmAdmin, adminID, email, role, "")
	if err != nil {
		env.t.Fatalf("RegisterAdmin: token: %v", err)
	}
	return token
}

// FakeUUID returns a random UUID string for test placeholders.
func FakeUUID() string {
	return uuid.New().String()
}

// StripeWebhookSignature generates a valid Stripe webhook signature for testing.
func StripeWebhookSignature(payload []byte) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	signedPayload := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(TestStripeWebhookSecret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%s,v1=%s", ts, sig)
}
