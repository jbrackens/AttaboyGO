package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
)

// PragmaticAdapter handles Pragmatic Play wallet server callbacks.
type PragmaticAdapter struct {
	secretKey string
	logger    *slog.Logger
}

// NewPragmaticAdapter creates a new Pragmatic Play adapter.
func NewPragmaticAdapter(secretKey string, logger *slog.Logger) *PragmaticAdapter {
	return &PragmaticAdapter{secretKey: secretKey, logger: logger}
}

// PragmaticRequest is the common request shape from Pragmatic Play.
type PragmaticRequest struct {
	UserID          string `json:"userId"`
	GameID          string `json:"gameId"`
	RoundID         string `json:"roundId"`
	TransactionID   string `json:"reference"`
	Amount          string `json:"amount"` // decimal string in currency units
	Currency        string `json:"currency"`
	ProvidedHash    string `json:"hash"`
	Token           string `json:"token"`
	Action          string `json:"action"` // authenticate, balance, bet, result, refund
}

// PragmaticResponse is the common response shape for Pragmatic Play.
type PragmaticResponse struct {
	Currency string `json:"currency"`
	Cash     string `json:"cash"` // decimal string
	Bonus    string `json:"bonus"`
	Error    int    `json:"error"`
	Message  string `json:"description,omitempty"`
}

// VerifySignature validates the HMAC-SHA256 hash for a Pragmatic request.
func (a *PragmaticAdapter) VerifySignature(body []byte, providedHash string) bool {
	mac := hmac.New(sha256.New, []byte(a.secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(providedHash))
}

// ParseRequest extracts a Pragmatic Play request from an HTTP request.
func (a *PragmaticAdapter) ParseRequest(r *http.Request) (*PragmaticRequest, []byte, error) {
	body, err := readBody(r, 1<<20)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	var req PragmaticRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, body, fmt.Errorf("unmarshal: %w", err)
	}

	return &req, body, nil
}

// PlayerID parses the player UUID from a Pragmatic request.
func (a *PragmaticAdapter) PlayerID(req *PragmaticRequest) (uuid.UUID, error) {
	return uuid.Parse(req.UserID)
}

// ToWalletCallback converts a Pragmatic request to a unified WalletCallback.
func (a *PragmaticAdapter) ToWalletCallback(req *PragmaticRequest) (*WalletCallback, error) {
	playerID, err := a.PlayerID(req)
	if err != nil {
		return nil, domain.ErrValidation("invalid user id")
	}

	var action WalletAction
	switch req.Action {
	case "balance":
		action = WalletActionBalance
	case "bet":
		action = WalletActionBet
	case "result":
		action = WalletActionWin
	case "refund":
		action = WalletActionRollback
	default:
		return nil, domain.ErrValidation(fmt.Sprintf("unknown action: %s", req.Action))
	}

	// Parse amount — Pragmatic sends decimal strings in currency units (e.g., "10.50" = 1050 cents)
	amountCents, err := parseDecimalToCents(req.Amount)
	if err != nil {
		return nil, domain.ErrValidation("invalid amount")
	}

	return &WalletCallback{
		Action:        action,
		PlayerID:      playerID,
		Amount:        amountCents,
		Currency:      req.Currency,
		TransactionID: req.TransactionID,
		RoundID:       req.RoundID,
		GameID:        req.GameID,
	}, nil
}

// RespondJSON writes a Pragmatic Play JSON response.
func (a *PragmaticAdapter) RespondJSON(w http.ResponseWriter, resp PragmaticResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Pragmatic always expects 200
	json.NewEncoder(w).Encode(resp)
}

// parseDecimalToCents converts "10.50" to 1050.
func parseDecimalToCents(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	var whole, frac int64
	parts := splitDecimal(s)

	_, err := fmt.Sscanf(parts[0], "%d", &whole)
	if err != nil {
		return 0, err
	}

	if len(parts) > 1 {
		fracStr := parts[1]
		// Pad or truncate to 2 decimal places
		for len(fracStr) < 2 {
			fracStr += "0"
		}
		fracStr = fracStr[:2]
		_, err = fmt.Sscanf(fracStr, "%d", &frac)
		if err != nil {
			return 0, err
		}
	}

	return whole*100 + frac, nil
}

func splitDecimal(s string) []string {
	for i, c := range s {
		if c == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// FormatCents converts cents to decimal string (1050 → "10.50").
func FormatCents(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	return fmt.Sprintf("%d.%02d", whole, frac)
}
