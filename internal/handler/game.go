package handler

import (
	"context"
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Game represents a game row for API responses.
type Game struct {
	ID             uuid.UUID `json:"id"`
	ManufacturerID string    `json:"manufacturer_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Category       string    `json:"category"`
	Provider       string    `json:"provider"`
	IsActive       bool      `json:"is_active"`
	RTP            *float64  `json:"rtp,omitempty"`
	Volatility     string    `json:"volatility,omitempty"`
	ThumbnailURL   string    `json:"thumbnail_url,omitempty"`
}

// GameRepository provides read access to games.
type GameRepository interface {
	List(ctx context.Context, db repository.DBTX, category string, limit, offset int) ([]Game, error)
	FindByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*Game, error)
}

// GameHandler handles game listing endpoints.
type GameHandler struct {
	games GameRepository
	db    repository.DBTX
}

// NewGameHandler creates a new GameHandler.
func NewGameHandler(games GameRepository, db repository.DBTX) *GameHandler {
	return &GameHandler{games: games, db: db}
}

// ListGames handles GET /games.
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	games, err := h.games.List(r.Context(), h.db, category, 50, 0)
	if err != nil {
		RespondError(w, domain.ErrInternal("list games", err))
		return
	}
	RespondJSON(w, http.StatusOK, games)
}

// GetGame handles GET /games/{id}.
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "VALIDATION_ERROR",
			"message": "invalid game id",
		})
		return
	}

	game, err := h.games.FindByID(r.Context(), h.db, id)
	if err != nil {
		RespondError(w, domain.ErrInternal("find game", err))
		return
	}
	if game == nil {
		RespondError(w, domain.ErrNotFound("game", id.String()))
		return
	}

	RespondJSON(w, http.StatusOK, game)
}
