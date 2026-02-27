package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

// BalanceUpdate describes which columns to update and by how much.
// Used by PostLedgerEntry to build the dynamic UPDATE statement (Audit #1).
type BalanceUpdate struct {
	Balance         int64 // delta for balance column
	BonusBalance    int64 // delta for bonus_balance column
	ReservedBalance int64 // delta for reserved_balance column
}

// HasBalanceDelta returns true if the real balance changes.
func (u BalanceUpdate) HasBalanceDelta() bool { return u.Balance != 0 }

// HasBonusDelta returns true if the bonus balance changes.
func (u BalanceUpdate) HasBonusDelta() bool { return u.BonusBalance != 0 }

// HasReservedDelta returns true if the reserved balance changes.
func (u BalanceUpdate) HasReservedDelta() bool { return u.ReservedBalance != 0 }

// PostLedgerEntryParams is the input to the atomic PostLedgerEntry operation.
type PostLedgerEntryParams struct {
	PlayerID              uuid.UUID
	Type                  TransactionType
	Amount                int64
	BalanceUpdate         BalanceUpdate
	ExternalTransactionID *string
	ManufacturerID        *string
	SubTransactionID      *string
	TargetTransactionID   *uuid.UUID
	GameRoundID           *string
	Metadata              json.RawMessage
}

// CommandResult is the return value from all 9 wallet commands.
type CommandResult struct {
	Transaction *Transaction
	Player      *Player
	Events      []OutboxDraft
	Idempotent  bool // true if this was a duplicate that returned existing tx
}

// DepositParams holds the input for ExecuteDeposit.
type DepositParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	ManufacturerID        string
	SubTransactionID      string
	Metadata              json.RawMessage
}

// PlaceBetParams holds the input for ExecutePlaceBet.
type PlaceBetParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	ManufacturerID        string
	SubTransactionID      string
	GameRoundID           string
	Metadata              json.RawMessage
}

// CreditWinParams holds the input for ExecuteCreditWin.
type CreditWinParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	ManufacturerID        string
	SubTransactionID      string
	GameRoundID           string
	WinType               CasinoWinType
	Metadata              json.RawMessage
}

// CancelTransactionParams holds the input for ExecuteCancelTransaction.
type CancelTransactionParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	ManufacturerID        string
	SubTransactionID      string
	TargetTransactionID   uuid.UUID
	Metadata              json.RawMessage
}

// WithdrawParams holds the input for ExecuteWithdraw.
type WithdrawParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	Metadata              json.RawMessage
}

// CompleteWithdrawalParams holds the input for ExecuteCompleteWithdrawal.
type CompleteWithdrawalParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	Metadata              json.RawMessage
}

// BonusCreditParams holds the input for ExecuteBonusCredit.
type BonusCreditParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	Metadata              json.RawMessage
}

// TurnBonusToRealParams holds the input for ExecuteTurnBonusToReal.
type TurnBonusToRealParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	Metadata              json.RawMessage
}

// ForfeitBonusParams holds the input for ExecuteForfeitBonus.
type ForfeitBonusParams struct {
	PlayerID              uuid.UUID
	Amount                int64
	ExternalTransactionID string
	IsBonusLost           bool // true → bonus_lost, false → bonus_forfeit
	Metadata              json.RawMessage
}
