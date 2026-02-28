package handler

import (
	"net/http"
	"strings"

	"github.com/attaboy/platform/internal/service"
	"github.com/go-chi/chi/v5"
)

// AffiliateHandler handles affiliate auth and portal endpoints.
type AffiliateHandler struct {
	affSvc *service.AffiliateService
}

// NewAffiliateHandler creates a new AffiliateHandler.
func NewAffiliateHandler(affSvc *service.AffiliateService) *AffiliateHandler {
	return &AffiliateHandler{affSvc: affSvc}
}

// Register handles POST /affiliates/register.
func (h *AffiliateHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.AffiliateRegisterInput
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	if strings.TrimSpace(input.Email) == "" {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "email is required",
		})
		return
	}
	if len(input.Password) < 8 {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "password must be at least 8 characters",
		})
		return
	}

	result, err := h.affSvc.Register(r.Context(), input)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, result)
}

// Login handles POST /affiliates/login.
func (h *AffiliateHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.AffiliateLoginInput
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	input.IP = ClientIP(r)

	result, err := h.affSvc.Login(r.Context(), input)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

// TrackClick handles GET /track/{btag} — public, non-authenticated.
func (h *AffiliateHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	btag := chi.URLParam(r, "btag")
	if btag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.affSvc.TrackClick(r.Context(), btag, r.RemoteAddr, r.UserAgent(), r.Referer())

	// Return 204 — tracking is async and non-blocking
	w.WriteHeader(http.StatusNoContent)
}
