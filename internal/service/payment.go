package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/policy"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentService handles deposit and withdrawal orchestration.
type PaymentService struct {
	pool     *pgxpool.Pool
	stripe   *provider.StripeProvider
	payments repository.PaymentRepository
	players  repository.PlayerRepository
	txRepo   repository.TransactionRepository
	engine   *ledger.Engine
	logger   *slog.Logger
}

// NewPaymentService creates a PaymentService.
func NewPaymentService(
	pool *pgxpool.Pool,
	stripe *provider.StripeProvider,
	payments repository.PaymentRepository,
	players repository.PlayerRepository,
	txRepo repository.TransactionRepository,
	engine *ledger.Engine,
	logger *slog.Logger,
) *PaymentService {
	return &PaymentService{
		pool:     pool,
		stripe:   stripe,
		payments: payments,
		players:  players,
		txRepo:   txRepo,
		engine:   engine,
		logger:   logger,
	}
}

// DepositSession holds the Stripe checkout session details.
type DepositSession struct {
	SessionID  string `json:"session_id"`
	SessionURL string `json:"session_url"`
	PaymentID  string `json:"payment_id"`
}

// InitiateDeposit creates a Stripe checkout session and records a pending payment.
func (s *PaymentService) InitiateDeposit(ctx context.Context, playerID uuid.UUID, amount int64, currency, successURL, cancelURL string) (*DepositSession, error) {
	if currency == "" {
		currency = "EUR"
	}

	// Responsible gaming: check daily deposit limit before hitting Stripe.
	dailyDeposits, err := s.txRepo.DailySumByType(ctx, s.pool, playerID, string(domain.TxDeposit))
	if err != nil {
		return nil, domain.ErrInternal("rg daily deposit query", err)
	}
	rgResult := policy.EvaluateRgLimits(policy.DefaultRgLimits(), amount, "wallet_deposit", dailyDeposits, 0)
	if !rgResult.Allowed {
		return nil, &domain.AppError{
			Code:    "RG_LIMIT_BREACHED",
			Message: fmt.Sprintf("deposit exceeds %s limit", rgResult.BreachedLimit),
			Status:  422,
		}
	}

	// Create Stripe checkout session
	session, err := s.stripe.CreateCheckoutSession(amount, currency, playerID.String(), successURL, cancelURL)
	if err != nil {
		return nil, domain.ErrInternal("create checkout session", err)
	}

	// Record pending payment
	providerName := "stripe"
	payment := &domain.Payment{
		ID:                uuid.New(),
		PlayerID:          playerID,
		Type:              domain.PaymentTypeDeposit,
		Amount:            amount,
		Currency:          currency,
		Status:            domain.PaymentStatusPending,
		Provider:          &providerName,
		ProviderSessionID: &session.ID,
	}
	if err := s.payments.Create(ctx, s.pool, payment); err != nil {
		return nil, domain.ErrInternal("record payment", err)
	}

	// Record payment event
	s.recordEvent(ctx, payment.ID, domain.PaymentStatusPending, "checkout session created", nil)

	return &DepositSession{
		SessionID:  session.ID,
		SessionURL: session.URL,
		PaymentID:  payment.ID.String(),
	}, nil
}

// HandleStripeWebhook processes a verified Stripe webhook event.
func (s *PaymentService) HandleStripeWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := s.stripe.VerifyWebhookSignature(payload, sigHeader)
	if err != nil {
		return domain.ErrUnauthorized(fmt.Sprintf("webhook verification failed: %v", err))
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	default:
		s.logger.Info("unhandled stripe event type", "type", event.Type)
		return nil
	}
}

func (s *PaymentService) handleCheckoutCompleted(ctx context.Context, event *provider.StripeWebhookEvent) error {
	sessionData, err := provider.ParseCheckoutSessionData(event.Data)
	if err != nil {
		return domain.ErrInternal("parse checkout session", err)
	}

	// Find the payment by provider session ID
	payment, err := s.payments.FindByProviderSessionID(ctx, s.pool, sessionData.ID)
	if err != nil {
		return domain.ErrInternal("find payment", err)
	}
	if payment == nil {
		s.logger.Warn("payment not found for session", "session_id", sessionData.ID)
		return nil // Don't error — Stripe may retry
	}

	// Idempotency: already completed
	if payment.Status == domain.PaymentStatusCompleted {
		return nil
	}

	// Credit the player's wallet via the ledger engine
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ErrInternal("begin tx", err)
	}
	defer tx.Rollback(ctx)

	extTxID := fmt.Sprintf("stripe_%s", event.ID)
	result, err := s.engine.ExecuteDeposit(ctx, tx, domain.DepositParams{
		PlayerID:              payment.PlayerID,
		Amount:                payment.Amount,
		ExternalTransactionID: extTxID,
		ManufacturerID:        "stripe",
		SubTransactionID:      "1",
		Metadata:              json.RawMessage(`{"provider":"stripe","event_id":"` + event.ID + `"}`),
	})
	if err != nil {
		return domain.ErrInternal("execute deposit", err)
	}

	// Update payment status
	ppID := sessionData.PaymentIntent
	if err := s.payments.UpdateStatus(ctx, tx, payment.ID, domain.PaymentStatusCompleted, &ppID, &result.Transaction.ID); err != nil {
		return domain.ErrInternal("update payment status", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ErrInternal("commit tx", err)
	}

	s.recordEvent(ctx, payment.ID, domain.PaymentStatusCompleted, "deposit credited via stripe", nil)
	s.logger.Info("deposit completed", "payment_id", payment.ID, "amount", payment.Amount, "player_id", payment.PlayerID)
	return nil
}

// RequestWithdrawal initiates a withdrawal (reserve balance, create pending withdrawal).
func (s *PaymentService) RequestWithdrawal(ctx context.Context, playerID uuid.UUID, amount int64) error {
	// Execute withdraw command (reserves balance)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ErrInternal("begin tx", err)
	}
	defer tx.Rollback(ctx)

	extTxID := fmt.Sprintf("wd_%s", uuid.New().String()[:8])
	_, err = s.engine.ExecuteWithdraw(ctx, tx, domain.WithdrawParams{
		PlayerID:              playerID,
		Amount:                amount,
		ExternalTransactionID: extTxID,
	})
	if err != nil {
		return err // Propagate domain errors (insufficient balance, etc.)
	}

	// Record pending withdrawal
	payment := &domain.Payment{
		ID:                    uuid.New(),
		PlayerID:              playerID,
		Type:                  domain.PaymentTypeWithdrawal,
		Amount:                amount,
		Currency:              "EUR",
		Status:                domain.PaymentStatusPending,
		ExternalTransactionID: &extTxID,
	}
	if err := s.payments.Create(ctx, tx, payment); err != nil {
		return domain.ErrInternal("record withdrawal", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ErrInternal("commit tx", err)
	}

	s.recordEvent(ctx, payment.ID, domain.PaymentStatusPending, "withdrawal requested", nil)
	return nil
}

// ListPayments returns a player's payment history.
func (s *PaymentService) ListPayments(ctx context.Context, playerID uuid.UUID) ([]domain.Payment, error) {
	payments, err := s.payments.ListByPlayer(ctx, s.pool, playerID, 50)
	if err != nil {
		return nil, domain.ErrInternal("list payments", err)
	}
	return payments, nil
}

func (s *PaymentService) recordEvent(ctx context.Context, paymentID uuid.UUID, status domain.PaymentStatus, message string, rawData json.RawMessage) {
	event := &domain.PaymentEvent{
		PaymentID: paymentID,
		Status:    status,
		Message:   &message,
		RawData:   rawData,
	}
	if err := s.payments.InsertEvent(ctx, s.pool, event); err != nil {
		s.logger.Error("record payment event", "error", err, "payment_id", paymentID)
	}
}
