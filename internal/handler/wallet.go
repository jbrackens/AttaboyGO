package handler

import (
	"net/http"
	"strconv"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
)

// WalletHandler handles wallet balance and transaction endpoints.
type WalletHandler struct {
	players      repository.PlayerRepository
	transactions repository.TransactionRepository
	db           repository.DBTX
}

// NewWalletHandler creates a new WalletHandler.
func NewWalletHandler(players repository.PlayerRepository, transactions repository.TransactionRepository, db repository.DBTX) *WalletHandler {
	return &WalletHandler{players: players, transactions: transactions, db: db}
}

// balanceResponse is the shape of GET /wallet/balance.
type balanceResponse struct {
	Balance         int64  `json:"balance"`
	BonusBalance    int64  `json:"bonus_balance"`
	ReservedBalance int64  `json:"reserved_balance"`
	Currency        string `json:"currency"`
}

// GetBalance handles GET /wallet/balance.
func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	player, err := h.players.FindByID(r.Context(), h.db, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("find player", err))
		return
	}
	if player == nil {
		RespondError(w, domain.ErrNotFound("player", playerID.String()))
		return
	}

	RespondJSON(w, http.StatusOK, balanceResponse{
		Balance:         player.Balance,
		BonusBalance:    player.BonusBalance,
		ReservedBalance: player.ReservedBalance,
		Currency:        player.Currency,
	})
}

// txListResponse wraps a list of transactions with cursor.
type txListResponse struct {
	Transactions []domain.Transaction `json:"transactions"`
	NextCursor   *string              `json:"next_cursor,omitempty"`
}

// GetTransactions handles GET /wallet/transactions with cursor-based pagination.
func (h *WalletHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	// Parse pagination params
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *string
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursor = &c
	}

	txs, err := h.transactions.ListByPlayer(r.Context(), h.db, playerID, cursor, limit+1)
	if err != nil {
		RespondError(w, domain.ErrInternal("list transactions", err))
		return
	}

	// Determine next cursor
	resp := txListResponse{Transactions: txs}
	if len(txs) > limit {
		resp.Transactions = txs[:limit]
		nextID := txs[limit].ID.String()
		resp.NextCursor = &nextID
	}

	RespondJSON(w, http.StatusOK, resp)
}

// playerIDFromContext extracts and validates the player UUID from auth context.
func playerIDFromContext(r *http.Request) (uuid.UUID, error) {
	sub := auth.SubjectFromContext(r.Context())
	if sub == "" {
		return uuid.Nil, domain.ErrUnauthorized("no subject in context")
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, domain.ErrUnauthorized("invalid subject")
	}
	return id, nil
}
