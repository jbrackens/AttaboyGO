package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VideoHandler handles video session endpoints.
type VideoHandler struct {
	pool *pgxpool.Pool
}

// NewVideoHandler creates a new VideoHandler.
func NewVideoHandler(pool *pgxpool.Pool) *VideoHandler {
	return &VideoHandler{pool: pool}
}

// StartSession handles POST /video/sessions.
func (h *VideoHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var input struct {
		StreamURL string `json:"stream_url"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	var sessionID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO video_sessions (player_id, stream_url)
		VALUES ($1, $2) RETURNING id`,
		playerID, input.StreamURL).Scan(&sessionID)
	if err != nil {
		RespondError(w, domain.ErrInternal("start video session", err))
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]string{"id": sessionID.String()})
}

// EndSession handles POST /video/sessions/{id}/end.
func (h *VideoHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid session id"))
		return
	}

	result, err := h.pool.Exec(r.Context(), `
		UPDATE video_sessions
		SET status = 'ended', ended_at = now(),
		    duration_minutes = EXTRACT(EPOCH FROM (now() - started_at))::integer / 60
		WHERE id = $1 AND player_id = $2 AND status = 'active'`,
		sessionID, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("end video session", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, domain.ErrNotFound("active video session", sessionID.String()))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"status": "ended"})
}

// ListSessions handles GET /video/sessions.
func (h *VideoHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, status, started_at, ended_at, duration_minutes
		FROM video_sessions
		WHERE player_id = $1
		ORDER BY started_at DESC LIMIT 20`, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("list video sessions", err))
		return
	}
	defer rows.Close()

	type session struct {
		ID              uuid.UUID  `json:"id"`
		Status          string     `json:"status"`
		StartedAt       time.Time  `json:"started_at"`
		EndedAt         *time.Time `json:"ended_at,omitempty"`
		DurationMinutes int        `json:"duration_minutes"`
	}

	var sessions []session
	for rows.Next() {
		var s session
		if err := rows.Scan(&s.ID, &s.Status, &s.StartedAt, &s.EndedAt, &s.DurationMinutes); err != nil {
			RespondError(w, domain.ErrInternal("scan video session", err))
			return
		}
		sessions = append(sessions, s)
	}

	RespondJSON(w, http.StatusOK, sessions)
}
