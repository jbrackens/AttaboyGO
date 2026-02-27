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

// QuestAdminHandler handles admin quest management.
type QuestAdminHandler struct {
	pool *pgxpool.Pool
}

// NewQuestAdminHandler creates a new QuestAdminHandler.
func NewQuestAdminHandler(pool *pgxpool.Pool) *QuestAdminHandler {
	return &QuestAdminHandler{pool: pool}
}

// ListQuests handles GET /admin/quests.
func (h *QuestAdminHandler) ListQuests(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name, description, type, target_progress, reward_amount, reward_currency,
		       min_score, cooldown_minutes, daily_budget_minor, active, sort_order, created_at
		FROM quests ORDER BY sort_order ASC`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list quests", err))
		return
	}
	defer rows.Close()

	type questRow struct {
		ID              uuid.UUID `json:"id"`
		Name            string    `json:"name"`
		Description     string    `json:"description"`
		Type            string    `json:"type"`
		TargetProgress  int       `json:"target_progress"`
		RewardAmount    int       `json:"reward_amount"`
		RewardCurrency  string    `json:"reward_currency"`
		MinScore        int       `json:"min_score"`
		CooldownMinutes int       `json:"cooldown_minutes"`
		DailyBudgetMinor int      `json:"daily_budget_minor"`
		Active          bool      `json:"active"`
		SortOrder       int       `json:"sort_order"`
		CreatedAt       time.Time `json:"created_at"`
	}

	var quests []questRow
	for rows.Next() {
		var q questRow
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &q.Type, &q.TargetProgress,
			&q.RewardAmount, &q.RewardCurrency, &q.MinScore, &q.CooldownMinutes,
			&q.DailyBudgetMinor, &q.Active, &q.SortOrder, &q.CreatedAt); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan quest", err))
			return
		}
		quests = append(quests, q)
	}

	handler.RespondJSON(w, http.StatusOK, quests)
}

// CreateQuest handles POST /admin/quests.
func (h *QuestAdminHandler) CreateQuest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		Type            string `json:"type"`
		TargetProgress  int    `json:"target_progress"`
		RewardAmount    int    `json:"reward_amount"`
		RewardCurrency  string `json:"reward_currency"`
		MinScore        int    `json:"min_score"`
		CooldownMinutes int    `json:"cooldown_minutes"`
		DailyBudgetMinor int   `json:"daily_budget_minor"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	var questID uuid.UUID
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO quests (name, description, type, target_progress, reward_amount, reward_currency,
			min_score, cooldown_minutes, daily_budget_minor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		input.Name, input.Description, input.Type, input.TargetProgress,
		input.RewardAmount, input.RewardCurrency, input.MinScore,
		input.CooldownMinutes, input.DailyBudgetMinor,
	).Scan(&questID)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("create quest", err))
		return
	}

	handler.RespondJSON(w, http.StatusCreated, map[string]string{"id": questID.String()})
}

// ToggleQuest handles PATCH /admin/quests/{id}/toggle.
func (h *QuestAdminHandler) ToggleQuest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid quest id"))
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE quests SET active = NOT active, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("toggle quest", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "toggled"})
}
