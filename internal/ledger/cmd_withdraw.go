package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteWithdraw moves funds from balance to reserved_balance (two-phase withdrawal).
// Phase 1: balance -= amount, reserved_balance += amount
func (e *Engine) ExecuteWithdraw(ctx context.Context, tx pgx.Tx, params domain.WithdrawParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("withdraw: %w", err)
	}

	// Idempotency check
	extID := params.ExternalTransactionID
	if extID != "" {
		existing, err := e.FindExistingTransaction(ctx, tx, domain.IdempotencyKey{
			PlayerID:              params.PlayerID,
			ExternalTransactionID: extID,
		})
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &domain.CommandResult{Transaction: existing, Player: player, Idempotent: true}, nil
		}
	}

	// Check sufficient real balance
	if player.Balance < params.Amount {
		return nil, domain.ErrInsufficientBalance()
	}

	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxWithdrawal,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{Balance: -params.Amount, ReservedBalance: params.Amount},
		ExternalTransactionID: strPtr(extID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("withdraw post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}
