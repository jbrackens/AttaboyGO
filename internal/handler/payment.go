package handler

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/service"
)

// PaymentHandler handles deposit and withdrawal endpoints.
type PaymentHandler struct {
	paymentSvc *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(paymentSvc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

type initiateDepositRequest struct {
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// InitiateDeposit handles POST /payments/deposit.
func (h *PaymentHandler) InitiateDeposit(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var req initiateDepositRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	if req.Amount <= 0 {
		RespondError(w, domain.ErrValidation("amount must be positive"))
		return
	}

	session, err := h.paymentSvc.InitiateDeposit(r.Context(), playerID, req.Amount, req.Currency, req.SuccessURL, req.CancelURL)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, session)
}

type requestWithdrawalRequest struct {
	Amount int64 `json:"amount"`
}

// RequestWithdrawal handles POST /payments/withdraw.
func (h *PaymentHandler) RequestWithdrawal(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var req requestWithdrawalRequest
	if err := DecodeJSON(r, &req); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	if req.Amount <= 0 {
		RespondError(w, domain.ErrValidation("amount must be positive"))
		return
	}

	if err := h.paymentSvc.RequestWithdrawal(r.Context(), playerID, req.Amount); err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

// GetPaymentHistory handles GET /payments/history.
func (h *PaymentHandler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	payments, err := h.paymentSvc.ListPayments(r.Context(), playerID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, payments)
}
