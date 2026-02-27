package handler

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/provider"
)

// RNGHandler handles random number and slot game endpoints.
type RNGHandler struct {
	rng      *provider.RandomOrgClient
	slotopol *provider.SlotopolClient
}

// NewRNGHandler creates a new RNGHandler.
func NewRNGHandler(rng *provider.RandomOrgClient, slotopol *provider.SlotopolClient) *RNGHandler {
	return &RNGHandler{rng: rng, slotopol: slotopol}
}

// GetRandom handles POST /rng/random — returns random integers.
func (h *RNGHandler) GetRandom(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Count int `json:"count"`
		Min   int `json:"min"`
		Max   int `json:"max"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	if input.Count <= 0 || input.Count > 100 {
		input.Count = 1
	}
	if input.Max <= input.Min {
		RespondError(w, domain.ErrValidation("max must be greater than min"))
		return
	}

	numbers, err := h.rng.RandomIntegers(r.Context(), input.Count, input.Min, input.Max)
	if err != nil {
		RespondError(w, domain.ErrInternal("generate random", err))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"numbers": numbers,
	})
}

// ListSlotGames handles GET /slots/games.
func (h *RNGHandler) ListSlotGames(w http.ResponseWriter, r *http.Request) {
	games, err := h.slotopol.ListGames(r.Context())
	if err != nil {
		RespondError(w, domain.ErrInternal("list slot games", err))
		return
	}

	RespondJSON(w, http.StatusOK, games)
}

// Spin handles POST /slots/spin.
func (h *RNGHandler) Spin(w http.ResponseWriter, r *http.Request) {
	_, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var input provider.SlotopolSpinRequest
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	result, err := h.slotopol.Spin(r.Context(), input)
	if err != nil {
		RespondError(w, domain.ErrInternal("slot spin", err))
		return
	}

	RespondJSON(w, http.StatusOK, result)
}
