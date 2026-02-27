package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/infra"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PaymentRepository provides access to the payments table.
type PaymentRepository interface {
	Create(ctx context.Context, db DBTX, payment *domain.Payment) error
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, db DBTX, id uuid.UUID, status domain.PaymentStatus, providerPaymentID *string, txID *uuid.UUID) error
	ListByPlayer(ctx context.Context, db DBTX, playerID uuid.UUID, limit int) ([]domain.Payment, error)
	FindByProviderSessionID(ctx context.Context, db DBTX, sessionID string) (*domain.Payment, error)
	InsertEvent(ctx context.Context, db DBTX, event *domain.PaymentEvent) error
}

type paymentRepo struct{}

// NewPaymentRepository returns a pgx-backed PaymentRepository.
func NewPaymentRepository() PaymentRepository {
	return &paymentRepo{}
}

func (r *paymentRepo) Create(ctx context.Context, db DBTX, p *domain.Payment) error {
	meta := p.Metadata
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}
	_, err := db.Exec(ctx, `
		INSERT INTO payments (id, player_id, type, amount, currency, status,
			payment_method_id, external_transaction_id, provider, provider_session_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		p.ID, p.PlayerID, string(p.Type),
		infra.Int64ToNumeric(p.Amount), p.Currency, string(p.Status),
		p.PaymentMethodID, p.ExternalTransactionID,
		p.Provider, p.ProviderSessionID, meta,
	)
	return err
}

func (r *paymentRepo) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Payment, error) {
	row := db.QueryRow(ctx, `
		SELECT id, player_id, type, amount, currency, status,
		       payment_method_id, external_transaction_id, transaction_id,
		       provider, provider_session_id, provider_payment_id,
		       approved_by, approved_at, metadata, created_at, updated_at
		FROM payments WHERE id = $1`, id)
	return scanPayment(row)
}

func (r *paymentRepo) FindByProviderSessionID(ctx context.Context, db DBTX, sessionID string) (*domain.Payment, error) {
	row := db.QueryRow(ctx, `
		SELECT id, player_id, type, amount, currency, status,
		       payment_method_id, external_transaction_id, transaction_id,
		       provider, provider_session_id, provider_payment_id,
		       approved_by, approved_at, metadata, created_at, updated_at
		FROM payments WHERE provider_session_id = $1`, sessionID)
	return scanPayment(row)
}

func (r *paymentRepo) UpdateStatus(ctx context.Context, db DBTX, id uuid.UUID, status domain.PaymentStatus, providerPaymentID *string, txID *uuid.UUID) error {
	_, err := db.Exec(ctx, `
		UPDATE payments SET status = $2, provider_payment_id = COALESCE($3, provider_payment_id),
			transaction_id = COALESCE($4, transaction_id), updated_at = now()
		WHERE id = $1`,
		id, string(status), providerPaymentID, txID)
	return err
}

func (r *paymentRepo) ListByPlayer(ctx context.Context, db DBTX, playerID uuid.UUID, limit int) ([]domain.Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.Query(ctx, `
		SELECT id, player_id, type, amount, currency, status,
		       payment_method_id, external_transaction_id, transaction_id,
		       provider, provider_session_id, provider_payment_id,
		       approved_by, approved_at, metadata, created_at, updated_at
		FROM payments WHERE player_id = $1
		ORDER BY created_at DESC LIMIT $2`, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("query payments: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		p, err := scanPaymentRow(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *p)
	}
	return payments, rows.Err()
}

func (r *paymentRepo) InsertEvent(ctx context.Context, db DBTX, event *domain.PaymentEvent) error {
	raw := event.RawData
	if raw == nil {
		raw = json.RawMessage(`{}`)
	}
	_, err := db.Exec(ctx, `
		INSERT INTO payment_events (payment_id, status, message, admin_user_id, raw_data)
		VALUES ($1, $2, $3, $4, $5)`,
		event.PaymentID, string(event.Status), event.Message, event.AdminUserID, raw)
	return err
}

func scanPayment(row pgx.Row) (*domain.Payment, error) {
	var p domain.Payment
	var amountNum pgtype.Numeric
	err := row.Scan(
		&p.ID, &p.PlayerID, &p.Type, &amountNum, &p.Currency, &p.Status,
		&p.PaymentMethodID, &p.ExternalTransactionID, &p.TransactionID,
		&p.Provider, &p.ProviderSessionID, &p.ProviderPaymentID,
		&p.ApprovedBy, &p.ApprovedAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan payment: %w", err)
	}
	var convErr error
	p.Amount, convErr = infra.NumericToInt64(amountNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert payment amount: %w", convErr)
	}
	return &p, nil
}

func scanPaymentRow(rows pgx.Rows) (*domain.Payment, error) {
	var p domain.Payment
	var amountNum pgtype.Numeric
	err := rows.Scan(
		&p.ID, &p.PlayerID, &p.Type, &amountNum, &p.Currency, &p.Status,
		&p.PaymentMethodID, &p.ExternalTransactionID, &p.TransactionID,
		&p.Provider, &p.ProviderSessionID, &p.ProviderPaymentID,
		&p.ApprovedBy, &p.ApprovedAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan payment row: %w", err)
	}
	var convErr error
	p.Amount, convErr = infra.NumericToInt64(amountNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert payment amount: %w", convErr)
	}
	return &p, nil
}
