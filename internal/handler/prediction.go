package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PredictionHandler handles prediction market endpoints.
type PredictionHandler struct {
	pool *pgxpool.Pool
}

// NewPredictionHandler creates a new PredictionHandler.
func NewPredictionHandler(pool *pgxpool.Pool) *PredictionHandler {
	return &PredictionHandler{pool: pool}
}

type predictionMarketResponse struct {
	ID         uuid.UUID   `json:"id"`
	Title      string      `json:"title"`
	Category   string      `json:"category"`
	Status     string      `json:"status"`
	CloseAt    *time.Time  `json:"close_at,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

// ListMarkets handles GET /predictions/markets.
func (h *PredictionHandler) ListMarkets(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, title, category, status, close_at, created_at
		FROM prediction_markets
		WHERE status IN ('open', 'closed')
		ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		RespondError(w, domain.ErrInternal("list prediction markets", err))
		return
	}
	defer rows.Close()

	var markets []predictionMarketResponse
	for rows.Next() {
		var m predictionMarketResponse
		if err := rows.Scan(&m.ID, &m.Title, &m.Category, &m.Status, &m.CloseAt, &m.CreatedAt); err != nil {
			RespondError(w, domain.ErrInternal("scan prediction market", err))
			return
		}
		markets = append(markets, m)
	}

	RespondJSON(w, http.StatusOK, markets)
}

// GetMarket handles GET /predictions/markets/{id}.
func (h *PredictionHandler) GetMarket(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid market id"))
		return
	}

	var m predictionMarketResponse
	err = h.pool.QueryRow(r.Context(), `
		SELECT id, title, category, status, close_at, created_at
		FROM prediction_markets WHERE id = $1`, id).
		Scan(&m.ID, &m.Title, &m.Category, &m.Status, &m.CloseAt, &m.CreatedAt)
	if err != nil {
		RespondError(w, domain.ErrNotFound("prediction market", id.String()))
		return
	}

	RespondJSON(w, http.StatusOK, m)
}

// PlaceStake handles POST /predictions/markets/{id}/stake.
func (h *PredictionHandler) PlaceStake(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid market id"))
		return
	}

	var input struct {
		OutcomeID string `json:"outcome_id"`
		Amount    int    `json:"amount"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	// Verify market is open
	var status string
	err = h.pool.QueryRow(r.Context(), `SELECT status FROM prediction_markets WHERE id = $1`, marketID).Scan(&status)
	if err != nil || status != "open" {
		RespondError(w, domain.ErrValidation("market is not open for stakes"))
		return
	}

	var stakeID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO prediction_stakes (player_id, market_id, outcome_id, stake_amount_minor)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		playerID, marketID, input.OutcomeID, input.Amount).Scan(&stakeID)
	if err != nil {
		RespondError(w, domain.ErrInternal("place stake", err))
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]string{"id": stakeID.String()})
}

// MyPositions handles GET /predictions/positions.
func (h *PredictionHandler) MyPositions(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT ps.id, ps.market_id, pm.title, ps.outcome_id, ps.stake_amount_minor, ps.status, ps.placed_at
		FROM prediction_stakes ps
		JOIN prediction_markets pm ON pm.id = ps.market_id
		WHERE ps.player_id = $1
		ORDER BY ps.placed_at DESC LIMIT 50`, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("list positions", err))
		return
	}
	defer rows.Close()

	type position struct {
		ID       uuid.UUID `json:"id"`
		MarketID uuid.UUID `json:"market_id"`
		Title    string    `json:"market_title"`
		Outcome  string    `json:"outcome_id"`
		Amount   int       `json:"stake_amount"`
		Status   string    `json:"status"`
		PlacedAt time.Time `json:"placed_at"`
	}

	var positions []position
	for rows.Next() {
		var p position
		if err := rows.Scan(&p.ID, &p.MarketID, &p.Title, &p.Outcome, &p.Amount, &p.Status, &p.PlacedAt); err != nil {
			RespondError(w, domain.ErrInternal("scan position", err))
			return
		}
		positions = append(positions, p)
	}

	RespondJSON(w, http.StatusOK, positions)
}
