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

// SportsbookAdminHandler handles admin sportsbook management.
type SportsbookAdminHandler struct {
	pool *pgxpool.Pool
}

// NewSportsbookAdminHandler creates a new SportsbookAdminHandler.
func NewSportsbookAdminHandler(pool *pgxpool.Pool) *SportsbookAdminHandler {
	return &SportsbookAdminHandler{pool: pool}
}

// CreateEvent handles POST /admin/sportsbook/events.
func (h *SportsbookAdminHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SportID   uuid.UUID `json:"sport_id"`
		League    string    `json:"league"`
		HomeTeam  string    `json:"home_team"`
		AwayTeam  string    `json:"away_team"`
		StartTime time.Time `json:"start_time"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	var eventID uuid.UUID
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO sports_events (sport_id, league, home_team, away_team, start_time)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		input.SportID, input.League, input.HomeTeam, input.AwayTeam, input.StartTime,
	).Scan(&eventID)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("create event", err))
		return
	}

	handler.RespondJSON(w, http.StatusCreated, map[string]string{"id": eventID.String()})
}

// UpdateEventStatus handles PATCH /admin/sportsbook/events/{id}/status.
func (h *SportsbookAdminHandler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid event id"))
		return
	}

	var input struct {
		Status    string `json:"status"`
		ScoreHome *int   `json:"score_home,omitempty"`
		ScoreAway *int   `json:"score_away,omitempty"`
	}
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE sports_events SET status = $2,
			score_home = COALESCE($3, score_home),
			score_away = COALESCE($4, score_away),
			updated_at = now()
		WHERE id = $1`,
		id, input.Status, input.ScoreHome, input.ScoreAway)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("update event", err))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ListEvents handles GET /admin/sportsbook/events.
func (h *SportsbookAdminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT e.id, e.sport_id, s.name as sport_name, e.league, e.home_team, e.away_team,
		       e.start_time, e.status, e.score_home, e.score_away
		FROM sports_events e JOIN sports s ON s.id = e.sport_id
		ORDER BY e.start_time DESC LIMIT 100`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list events", err))
		return
	}
	defer rows.Close()

	type eventSummary struct {
		ID        uuid.UUID `json:"id"`
		SportID   uuid.UUID `json:"sport_id"`
		SportName string    `json:"sport_name"`
		League    string    `json:"league"`
		HomeTeam  string    `json:"home_team"`
		AwayTeam  string    `json:"away_team"`
		StartTime time.Time `json:"start_time"`
		Status    string    `json:"status"`
		ScoreHome int       `json:"score_home"`
		ScoreAway int       `json:"score_away"`
	}

	var events []eventSummary
	for rows.Next() {
		var e eventSummary
		if err := rows.Scan(&e.ID, &e.SportID, &e.SportName, &e.League, &e.HomeTeam, &e.AwayTeam, &e.StartTime, &e.Status, &e.ScoreHome, &e.ScoreAway); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan event", err))
			return
		}
		events = append(events, e)
	}

	handler.RespondJSON(w, http.StatusOK, events)
}
