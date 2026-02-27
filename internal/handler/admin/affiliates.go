package admin

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AffiliateAdminHandler handles admin affiliate management.
type AffiliateAdminHandler struct {
	pool *pgxpool.Pool
}

// NewAffiliateAdminHandler creates a new AffiliateAdminHandler.
func NewAffiliateAdminHandler(pool *pgxpool.Pool) *AffiliateAdminHandler {
	return &AffiliateAdminHandler{pool: pool}
}

// ListAffiliates handles GET /admin/affiliates.
func (h *AffiliateAdminHandler) ListAffiliates(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, email, first_name, last_name, company, status, affiliate_code, tier, created_at
		FROM affiliates ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list affiliates", err))
		return
	}
	defer rows.Close()

	type affiliateSummary struct {
		ID            uuid.UUID `json:"id"`
		Email         string    `json:"email"`
		FirstName     string    `json:"first_name"`
		LastName      string    `json:"last_name"`
		Company       string    `json:"company"`
		Status        string    `json:"status"`
		AffiliateCode string    `json:"affiliate_code"`
		Tier          string    `json:"tier"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var affiliates []affiliateSummary
	for rows.Next() {
		var a affiliateSummary
		if err := rows.Scan(&a.ID, &a.Email, &a.FirstName, &a.LastName, &a.Company, &a.Status, &a.AffiliateCode, &a.Tier, &a.CreatedAt); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan affiliate", err))
			return
		}
		affiliates = append(affiliates, a)
	}

	handler.RespondJSON(w, http.StatusOK, affiliates)
}

// UpdateAffiliateStatus handles PATCH /admin/affiliates/{id}/status.
func (h *AffiliateAdminHandler) UpdateAffiliateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid affiliate id"))
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE affiliates SET status = $2, updated_at = now() WHERE id = $1`,
		id, input.Status)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("update affiliate status", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
