package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/attaboy/platform/internal/service"
)

// WebhookHandler handles Stripe webhook callbacks.
type WebhookHandler struct {
	paymentSvc *service.PaymentService
	logger     *slog.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(paymentSvc *service.PaymentService, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{paymentSvc: paymentSvc, logger: logger}
}

// HandleStripeWebhook handles POST /webhooks/stripe.
// IMPORTANT: This handler must receive the raw request body (no JSON middleware parsing).
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body (required for signature verification)
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		h.logger.Error("read webhook body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		h.logger.Warn("missing Stripe-Signature header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.paymentSvc.HandleStripeWebhook(r.Context(), body, sigHeader); err != nil {
		h.logger.Error("process stripe webhook", "error", err)
		RespondError(w, err)
		return
	}

	// Stripe expects 200 OK
	w.WriteHeader(http.StatusOK)
}
