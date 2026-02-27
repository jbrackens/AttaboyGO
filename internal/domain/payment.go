package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PaymentType distinguishes deposits from withdrawals.
type PaymentType string

const (
	PaymentTypeDeposit    PaymentType = "deposit"
	PaymentTypeWithdrawal PaymentType = "withdrawal"
)

// PaymentStatus tracks the payment lifecycle.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusApproved  PaymentStatus = "approved"
	PaymentStatusRejected  PaymentStatus = "rejected"
)

// Payment represents a payments table row.
type Payment struct {
	ID                    uuid.UUID       `json:"id"`
	PlayerID              uuid.UUID       `json:"player_id"`
	Type                  PaymentType     `json:"type"`
	Amount                int64           `json:"amount"`
	Currency              string          `json:"currency"`
	Status                PaymentStatus   `json:"status"`
	PaymentMethodID       *uuid.UUID      `json:"payment_method_id,omitempty"`
	ExternalTransactionID *string         `json:"external_transaction_id,omitempty"`
	TransactionID         *uuid.UUID      `json:"transaction_id,omitempty"`
	Provider              *string         `json:"provider,omitempty"`
	ProviderSessionID     *string         `json:"provider_session_id,omitempty"`
	ProviderPaymentID     *string         `json:"provider_payment_id,omitempty"`
	ApprovedBy            *uuid.UUID      `json:"approved_by,omitempty"`
	ApprovedAt            *time.Time      `json:"approved_at,omitempty"`
	Metadata              json.RawMessage `json:"metadata"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// PaymentEvent tracks status changes for audit trail.
type PaymentEvent struct {
	ID          uuid.UUID       `json:"id"`
	PaymentID   uuid.UUID       `json:"payment_id"`
	Status      PaymentStatus   `json:"status"`
	Message     *string         `json:"message,omitempty"`
	AdminUserID *uuid.UUID      `json:"admin_user_id,omitempty"`
	RawData     json.RawMessage `json:"raw_data,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}
