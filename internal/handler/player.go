package handler

import (
	"net/http"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
)

// PlayerHandler handles player profile endpoints.
type PlayerHandler struct {
	players  repository.PlayerRepository
	profiles repository.ProfileRepository
	db       repository.DBTX
}

// NewPlayerHandler creates a new PlayerHandler.
func NewPlayerHandler(players repository.PlayerRepository, profiles repository.ProfileRepository, db repository.DBTX) *PlayerHandler {
	return &PlayerHandler{players: players, profiles: profiles, db: db}
}

// meResponse combines player balance + profile for GET /players/me.
type meResponse struct {
	PlayerID      uuid.UUID       `json:"player_id"`
	Email         string          `json:"email"`
	Balance       domain.Balances `json:"balance"`
	Currency      string          `json:"currency"`
	AccountStatus string          `json:"account_status"`
	FirstName     string          `json:"first_name,omitempty"`
	LastName      string          `json:"last_name,omitempty"`
	Verified      bool            `json:"verified"`
}

// GetMe handles GET /players/me — returns current player's profile + balance.
func (h *PlayerHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	sub := auth.SubjectFromContext(r.Context())
	if sub == "" {
		RespondError(w, domain.ErrUnauthorized("no subject in context"))
		return
	}

	playerID, err := uuid.Parse(sub)
	if err != nil {
		RespondError(w, domain.ErrUnauthorized("invalid subject"))
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

	profile, err := h.profiles.FindByPlayerID(r.Context(), h.db, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("find profile", err))
		return
	}

	resp := meResponse{
		PlayerID: player.ID,
		Balance:  player.Balances,
		Currency: player.Currency,
	}
	if profile != nil {
		resp.Email = profile.Email
		resp.AccountStatus = profile.AccountStatus
		resp.FirstName = profile.FirstName
		resp.LastName = profile.LastName
		resp.Verified = profile.Verified
	}

	RespondJSON(w, http.StatusOK, resp)
}
