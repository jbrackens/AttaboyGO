package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EngagementHandler handles engagement tracking endpoints.
type EngagementHandler struct {
	pool *pgxpool.Pool
}

// NewEngagementHandler creates a new EngagementHandler.
func NewEngagementHandler(pool *pgxpool.Pool) *EngagementHandler {
	return &EngagementHandler{pool: pool}
}

type engagementResponse struct {
	Date               string `json:"date"`
	VideoMinutes       int    `json:"video_minutes"`
	SocialInteractions int    `json:"social_interactions"`
	PredictionActions  int    `json:"prediction_actions"`
	WagerCount         int    `json:"wager_count"`
	DepositCount       int    `json:"deposit_count"`
	Score              int    `json:"score"`
}

// GetMyEngagement handles GET /engagement/me.
func (h *EngagementHandler) GetMyEngagement(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	today := time.Now().Format("2006-01-02")
	var resp engagementResponse
	var dateVal time.Time
	err = h.pool.QueryRow(r.Context(), `
		SELECT date, video_minutes, social_interactions, prediction_actions,
		       wager_count, deposit_count, score
		FROM player_engagement WHERE player_id = $1 AND date = $2`,
		playerID, today).Scan(
		&dateVal, &resp.VideoMinutes, &resp.SocialInteractions,
		&resp.PredictionActions, &resp.WagerCount, &resp.DepositCount, &resp.Score)
	if err != nil {
		// No engagement today — return zeros
		resp.Date = today
	} else {
		resp.Date = dateVal.Format("2006-01-02")
	}

	RespondJSON(w, http.StatusOK, resp)
}

// RecordSignal handles POST /engagement/signal — records an engagement event.
func (h *EngagementHandler) RecordSignal(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var input struct {
		Type  string `json:"type"`
		Value int    `json:"value"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	today := time.Now().Format("2006-01-02")

	// Upsert engagement for today
	var colName, colExpr string
	switch input.Type {
	case "video":
		colName = "video_minutes"
		colExpr = "video_minutes = player_engagement.video_minutes + $3"
	case "social":
		colName = "social_interactions"
		colExpr = "social_interactions = player_engagement.social_interactions + $3"
	case "prediction":
		colName = "prediction_actions"
		colExpr = "prediction_actions = player_engagement.prediction_actions + $3"
	case "wager":
		colName = "wager_count"
		colExpr = "wager_count = player_engagement.wager_count + $3"
	case "deposit":
		colName = "deposit_count"
		colExpr = "deposit_count = player_engagement.deposit_count + $3"
	default:
		RespondError(w, domain.ErrValidation("invalid signal type"))
		return
	}

	// Use upsert to create or update daily engagement
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO player_engagement (player_id, date, `+colName+`, score)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (player_id, date) DO UPDATE SET `+colExpr+`, updated_at = now()`,
		playerID, today, input.Value)
	if err != nil {
		// Fallback: just update if the column naming doesn't match exactly
		RespondError(w, domain.ErrInternal("record signal", err))
		return
	}

	// Recompute score: video*2 + social*3 + prediction*5
	_, err = h.pool.Exec(r.Context(), `
		UPDATE player_engagement SET score = (video_minutes * 2 + social_interactions * 3 + prediction_actions * 5)
		WHERE player_id = $1 AND date = $2`, playerID, today)
	if err != nil {
		RespondError(w, domain.ErrInternal("update score", err))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}
