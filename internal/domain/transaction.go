package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TransactionType enumerates all wallet transaction types.
type TransactionType string

const (
	// Core
	TxDeposit             TransactionType = "wallet_deposit"
	TxWithdrawal          TransactionType = "wallet_withdrawal"
	TxWithdrawalProcessed TransactionType = "wallet_withdrawal_processed"
	TxBet                 TransactionType = "bet"
	TxWin                 TransactionType = "win"
	TxSettlementLoss      TransactionType = "settlement_loss"

	// Cancellation
	TxCancelDeposit     TransactionType = "cancel_deposit"
	TxCancelBet         TransactionType = "cancel_bet"
	TxCancelWin         TransactionType = "cancel_win"
	TxCancelWithdrawal  TransactionType = "wallet_cancel_withdrawal"

	// Bonus
	TxBonusCredit      TransactionType = "bonus_credit"
	TxBonusForfeit     TransactionType = "bonus_forfeit"
	TxBonusLost        TransactionType = "bonus_lost"
	TxTurnBonusToReal  TransactionType = "turn_bonus_to_real"
)

// CancellationTypeMap maps original transaction types to their cancel type.
var CancellationTypeMap = map[TransactionType]TransactionType{
	TxDeposit:    TxCancelDeposit,
	TxBet:        TxCancelBet,
	TxWin:        TxCancelWin,
	TxWithdrawal: TxCancelWithdrawal,
}

// CasinoWinType specifies sub-types of casino wins stored in metadata.
type CasinoWinType string

const (
	CasinoWinNormal       CasinoWinType = "win"
	CasinoWinJackpot      CasinoWinType = "win_jackpot"
	CasinoWinLocalJackpot CasinoWinType = "win_local_jackpot"
	CasinoWinFreespins    CasinoWinType = "win_freespins"
)

// Transaction represents a v2_transactions row (append-only ledger entry).
type Transaction struct {
	ID                    uuid.UUID       `json:"id"`
	PlayerID              uuid.UUID       `json:"player_id"`
	Type                  TransactionType `json:"type"`
	Amount                int64           `json:"amount"`
	BalanceAfter          int64           `json:"balance_after"`
	BonusBalanceAfter     int64           `json:"bonus_balance_after"`
	ReservedBalanceAfter  int64           `json:"reserved_balance_after"`
	ExternalTransactionID *string         `json:"external_transaction_id,omitempty"`
	ManufacturerID        *string         `json:"manufacturer_id,omitempty"`
	SubTransactionID      *string         `json:"sub_transaction_id,omitempty"`
	TargetTransactionID   *uuid.UUID      `json:"target_transaction_id,omitempty"`
	GameRoundID           *string         `json:"game_round_id,omitempty"`
	Metadata              json.RawMessage `json:"metadata"`
	CreatedAt             time.Time       `json:"created_at"`
}

// IdempotencyKey is the composite key used for deduplication.
type IdempotencyKey struct {
	PlayerID              uuid.UUID
	ManufacturerID        string
	ExternalTransactionID string
	SubTransactionID      string
}
