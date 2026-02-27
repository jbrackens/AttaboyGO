package admin

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BonusAdminHandler handles admin bonus management.
type BonusAdminHandler struct {
	pool *pgxpool.Pool
}

// NewBonusAdminHandler creates a new BonusAdminHandler.
func NewBonusAdminHandler(pool *pgxpool.Pool) *BonusAdminHandler {
	return &BonusAdminHandler{pool: pool}
}

// ListBonuses handles GET /admin/bonuses.
func (h *BonusAdminHandler) ListBonuses(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name, code, wagering_multiplier, min_deposit, max_bonus,
		       days_until_expiry, active
		FROM bonuses ORDER BY active DESC, name ASC LIMIT 50`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list bonuses", err))
		return
	}
	defer rows.Close()

	var bonuses []domain.Bonus
	for rows.Next() {
		var b domain.Bonus
		if err := rows.Scan(&b.ID, &b.Name, &b.Code, &b.WageringMultiplier, &b.MinDeposit, &b.MaxBonus, &b.DaysUntilExpiry, &b.Active); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan bonus", err))
			return
		}
		bonuses = append(bonuses, b)
	}

	handler.RespondJSON(w, http.StatusOK, bonuses)
}

// CreateBonus handles POST /admin/bonuses.
func (h *BonusAdminHandler) CreateBonus(w http.ResponseWriter, r *http.Request) {
	var input domain.Bonus
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	var bonusID uuid.UUID
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO bonuses (name, code, wagering_multiplier, min_deposit, max_bonus, days_until_expiry, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		input.Name, input.Code, input.WageringMultiplier, input.MinDeposit,
		input.MaxBonus, input.DaysUntilExpiry, true,
	).Scan(&bonusID)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("create bonus", err))
		return
	}

	handler.RespondJSON(w, http.StatusCreated, map[string]string{"id": bonusID.String()})
}

// UpdateBonusStatus handles PATCH /admin/bonuses/{id}/status.
func (h *BonusAdminHandler) UpdateBonusStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid bonus id"))
		return
	}

	var input struct {
		Active bool `json:"active"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	_, err = h.pool.Exec(r.Context(), `UPDATE bonuses SET active = $2 WHERE id = $1`, id, input.Active)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("update bonus", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
