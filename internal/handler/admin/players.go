package admin

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/handler"
	"github.com/attaboy/platform/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayerAdminHandler handles admin player management.
type PlayerAdminHandler struct {
	pool     *pgxpool.Pool
	players  repository.PlayerRepository
	profiles repository.ProfileRepository
}

// NewPlayerAdminHandler creates a new PlayerAdminHandler.
func NewPlayerAdminHandler(pool *pgxpool.Pool, players repository.PlayerRepository, profiles repository.ProfileRepository) *PlayerAdminHandler {
	return &PlayerAdminHandler{pool: pool, players: players, profiles: profiles}
}

// SearchPlayers handles GET /admin/players?q=email.
func (h *PlayerAdminHandler) SearchPlayers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	rows, err := h.pool.Query(r.Context(), `
		SELECT pp.player_id, pp.email, pp.first_name, pp.last_name, pp.account_status, pp.verified, pp.created_at,
		       p.balance, p.bonus_balance, p.reserved_balance, p.currency
		FROM player_profiles pp
		JOIN v2_players p ON p.id = pp.player_id
		WHERE ($1 = '' OR pp.email ILIKE '%' || $1 || '%')
		ORDER BY pp.created_at DESC LIMIT 50`, query)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("search players", err))
		return
	}
	defer rows.Close()

	type playerSummary struct {
		PlayerID      uuid.UUID `json:"player_id"`
		Email         string    `json:"email"`
		FirstName     string    `json:"first_name"`
		LastName      string    `json:"last_name"`
		AccountStatus string    `json:"account_status"`
		Verified      bool      `json:"verified"`
		CreatedAt     string    `json:"created_at"`
		Currency      string    `json:"currency"`
	}

	var results []playerSummary
	for rows.Next() {
		var ps playerSummary
		var bal, bonus, reserved interface{} // skip balance numerics for summary
		if err := rows.Scan(&ps.PlayerID, &ps.Email, &ps.FirstName, &ps.LastName, &ps.AccountStatus, &ps.Verified, &ps.CreatedAt, &bal, &bonus, &reserved, &ps.Currency); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan player", err))
			return
		}
		results = append(results, ps)
	}

	handler.RespondJSON(w, http.StatusOK, results)
}

// GetPlayerDetail handles GET /admin/players/{id}.
func (h *PlayerAdminHandler) GetPlayerDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid player id"))
		return
	}

	player, err := h.players.FindByID(r.Context(), h.pool, id)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("find player", err))
		return
	}
	if player == nil {
		handler.RespondError(w, domain.ErrNotFound("player", id.String()))
		return
	}

	profile, err := h.profiles.FindByPlayerID(r.Context(), h.pool, id)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("find profile", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"player":  player,
		"profile": profile,
	})
}

// UpdatePlayerStatus handles PATCH /admin/players/{id}/status.
func (h *PlayerAdminHandler) UpdatePlayerStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid player id"))
		return
	}

	var input struct {
		AccountStatus string `json:"account_status"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE player_profiles SET account_status = $2 WHERE player_id = $1`,
		id, input.AccountStatus)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("update status", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
