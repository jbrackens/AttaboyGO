package provider

import (
	"context"
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

// BetSolutionsAdapter handles BetSolutions wallet server callbacks.
type BetSolutionsAdapter struct {
	hmacSecret string
	logger     *slog.Logger
}

// NewBetSolutionsAdapter creates a new BetSolutions adapter.
func NewBetSolutionsAdapter(hmacSecret string, logger *slog.Logger) *BetSolutionsAdapter {
	return &BetSolutionsAdapter{hmacSecret: hmacSecret, logger: logger}
}

// BetSolutionsRequest is the common request shape from BetSolutions.
type BetSolutionsRequest struct {
	Token         string `json:"Token"`
	PlayerID      string `json:"PlayerId"`
	GameID        string `json:"GameId"`
	RoundID       string `json:"RoundId"`
	TransactionID string `json:"TransactionId"`
	Amount        int64  `json:"Amount"` // in cents
	Currency      string `json:"Currency"`
	Hash          string `json:"Hash"`
}

// BetSolutionsResponse is the common response shape for BetSolutions.
type BetSolutionsResponse struct {
	StatusCode int    `json:"StatusCode"`
	Balance    int64  `json:"Balance"`
	Error      string `json:"Error,omitempty"`
}

// VerifySignature validates the HMAC-SHA256 hash for a BetSolutions request.
// The signature is computed over the body with the Hash field excluded.
func (a *BetSolutionsAdapter) VerifySignature(body []byte, providedHash string) bool {
	stripped := stripJSONField(body, "Hash")
	mac := hmac.New(sha256.New, []byte(a.hmacSecret))
	mac.Write(stripped)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(providedHash))
}

// ComputeSignature computes the HMAC-SHA256 hash for a body with the given field excluded.
func (a *BetSolutionsAdapter) ComputeSignature(body []byte) string {
	stripped := stripJSONField(body, "Hash")
	mac := hmac.New(sha256.New, []byte(a.hmacSecret))
	mac.Write(stripped)
	return hex.EncodeToString(mac.Sum(nil))
}

// ParseRequest extracts and validates a BetSolutions request from an HTTP request.
func (a *BetSolutionsAdapter) ParseRequest(r *http.Request) (*BetSolutionsRequest, []byte, error) {
	body, err := readBody(r, 1<<20) // 1MB
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	var req BetSolutionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, body, fmt.Errorf("unmarshal: %w", err)
	}

	return &req, body, nil
}

// PlayerID parses the player UUID from a BetSolutions request.
func (a *BetSolutionsAdapter) PlayerID(req *BetSolutionsRequest) (uuid.UUID, error) {
	return uuid.Parse(req.PlayerID)
}

// ToCents converts provider amount to internal cents (BetSolutions sends cents).
func ToCents(amount int64) int64 {
	return amount
}

// FromCents converts internal cents to provider amount.
func FromCents(amount int64) int64 {
	return amount
}

// WalletAction represents the type of wallet callback.
type WalletAction string

const (
	WalletActionBalance  WalletAction = "balance"
	WalletActionBet      WalletAction = "bet"
	WalletActionWin      WalletAction = "win"
	WalletActionRollback WalletAction = "rollback"
)

// WalletCallback is the unified interface for game provider wallet operations.
type WalletCallback struct {
	Action        WalletAction
	PlayerID      uuid.UUID
	Amount        int64
	Currency      string
	TransactionID string
	RoundID       string
	GameID        string
}

// ToWalletCallback converts a BetSolutions request to a unified WalletCallback.
func (a *BetSolutionsAdapter) ToWalletCallback(req *BetSolutionsRequest, action WalletAction) (*WalletCallback, error) {
	playerID, err := a.PlayerID(req)
	if err != nil {
		return nil, domain.ErrValidation("invalid player id")
	}

	return &WalletCallback{
		Action:        action,
		PlayerID:      playerID,
		Amount:        ToCents(req.Amount),
		Currency:      req.Currency,
		TransactionID: req.TransactionID,
		RoundID:       req.RoundID,
		GameID:        req.GameID,
	}, nil
}

// RespondJSON writes a BetSolutions JSON response.
func (a *BetSolutionsAdapter) RespondJSON(w http.ResponseWriter, resp BetSolutionsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // BetSolutions always expects 200
	json.NewEncoder(w).Encode(resp)
}

func readBody(r *http.Request, maxSize int64) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)
	return readAll(r.Body)
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return result, err
		}
	}
	return result, nil
}

// stripJSONField removes a top-level field from a JSON body by unmarshalling,
// deleting the key, and re-marshalling. Used to exclude signature fields before
// computing HMAC.
func stripJSONField(body []byte, field string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return body // fallback to original if not valid JSON
	}
	delete(m, field)
	stripped, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return stripped
}

// contextKey for wallet operations
type contextKey string

const walletProviderKey contextKey = "wallet_provider"

// WithProvider sets the provider name in context.
func WithProvider(ctx context.Context, provider string) context.Context {
	return context.WithValue(ctx, walletProviderKey, provider)
}

// ProviderFromContext extracts the provider name from context.
func ProviderFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(walletProviderKey).(string); ok {
		return v
	}
	return ""
}
