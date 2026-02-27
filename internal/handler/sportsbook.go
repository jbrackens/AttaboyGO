package handler

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/attaboy/platform/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SportsbookHandler handles sportsbook endpoints.
type SportsbookHandler struct {
	svc *service.SportsbookService
	db  repository.DBTX
}

// NewSportsbookHandler creates a new SportsbookHandler.
func NewSportsbookHandler(svc *service.SportsbookService, db repository.DBTX) *SportsbookHandler {
	return &SportsbookHandler{svc: svc, db: db}
}

// ListSports handles GET /sportsbook/sports.
func (h *SportsbookHandler) ListSports(w http.ResponseWriter, r *http.Request) {
	sports, err := h.svc.ListSports(r.Context(), h.db)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, sports)
}

// ListEvents handles GET /sportsbook/sports/{sportID}/events.
func (h *SportsbookHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	sportID, err := uuid.Parse(chi.URLParam(r, "sportID"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid sport id"))
		return
	}
	events, err := h.svc.ListEvents(r.Context(), h.db, sportID)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, events)
}

// ListMarkets handles GET /sportsbook/events/{eventID}/markets.
func (h *SportsbookHandler) ListMarkets(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid event id"))
		return
	}
	markets, err := h.svc.ListMarkets(r.Context(), h.db, eventID)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, markets)
}

// ListSelections handles GET /sportsbook/markets/{marketID}/selections.
func (h *SportsbookHandler) ListSelections(w http.ResponseWriter, r *http.Request) {
	marketID, err := uuid.Parse(chi.URLParam(r, "marketID"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid market id"))
		return
	}
	selections, err := h.svc.ListSelections(r.Context(), h.db, marketID)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, selections)
}

// PlaceBet handles POST /sportsbook/bets.
func (h *SportsbookHandler) PlaceBet(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var input service.PlaceBetInput
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	result, err := h.svc.PlaceBet(r.Context(), playerID, input)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, result)
}

// MyBets handles GET /sportsbook/bets/me.
func (h *SportsbookHandler) MyBets(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	bets, err := h.svc.ListPlayerBets(r.Context(), playerID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, bets)
}
