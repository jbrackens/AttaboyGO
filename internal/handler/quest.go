package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuestHandler handles quest endpoints.
type QuestHandler struct {
	pool *pgxpool.Pool
}

// NewQuestHandler creates a new QuestHandler.
func NewQuestHandler(pool *pgxpool.Pool) *QuestHandler {
	return &QuestHandler{pool: pool}
}

type questWithProgress struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	TargetProgress  int       `json:"target_progress"`
	RewardAmount    int       `json:"reward_amount"`
	RewardCurrency  string    `json:"reward_currency"`
	MinScore        int       `json:"min_score"`
	Progress        int       `json:"progress"`
	Status          string    `json:"status"`
}

// ListActive handles GET /quests — returns active quests with player progress.
func (h *QuestHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT q.id, q.name, q.description, q.type, q.target_progress,
		       q.reward_amount, q.reward_currency, q.min_score,
		       COALESCE(pqp.progress, 0), COALESCE(pqp.status, 'not_started')
		FROM quests q
		LEFT JOIN player_quest_progress pqp ON pqp.quest_id = q.id AND pqp.player_id = $1
		WHERE q.active = true
		ORDER BY q.sort_order ASC`, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("query quests", err))
		return
	}
	defer rows.Close()

	var quests []questWithProgress
	for rows.Next() {
		var q questWithProgress
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &q.Type, &q.TargetProgress,
			&q.RewardAmount, &q.RewardCurrency, &q.MinScore, &q.Progress, &q.Status); err != nil {
			RespondError(w, domain.ErrInternal("scan quest", err))
			return
		}
		quests = append(quests, q)
	}

	RespondJSON(w, http.StatusOK, quests)
}

// ClaimReward handles POST /quests/{id}/claim.
func (h *QuestHandler) ClaimReward(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	// Check quest progress is completed and unclaimed
	var questID uuid.UUID
	var progressStatus string
	var rewardAmount int
	var rewardCurrency string

	err = h.pool.QueryRow(r.Context(), `
		SELECT pqp.quest_id, pqp.status, q.reward_amount, q.reward_currency
		FROM player_quest_progress pqp
		JOIN quests q ON q.id = pqp.quest_id
		WHERE pqp.player_id = $1 AND pqp.status = 'completed'
		LIMIT 1`, playerID).Scan(&questID, &progressStatus, &rewardAmount, &rewardCurrency)
	if err != nil {
		RespondError(w, domain.ErrNotFound("completed quest", playerID.String()))
		return
	}

	// Mark as claimed
	_, err = h.pool.Exec(r.Context(), `
		UPDATE player_quest_progress SET status = 'claimed', claimed_at = $2
		WHERE player_id = $1 AND quest_id = $3`,
		playerID, time.Now(), questID)
	if err != nil {
		RespondError(w, domain.ErrInternal("claim quest", err))
		return
	}

	// Record reward grant
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO reward_grants (player_id, quest_id, amount, currency)
		VALUES ($1, $2, $3, $4)`,
		playerID, questID, rewardAmount, rewardCurrency)
	if err != nil {
		RespondError(w, domain.ErrInternal("record reward", err))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"quest_id":        questID,
		"reward_amount":   rewardAmount,
		"reward_currency": rewardCurrency,
	})
}
