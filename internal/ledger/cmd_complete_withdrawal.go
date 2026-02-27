package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteCompleteWithdrawal finalizes a withdrawal by releasing reserved funds.
// Phase 2: reserved_balance -= amount
func (e *Engine) ExecuteCompleteWithdrawal(ctx context.Context, tx pgx.Tx, params domain.CompleteWithdrawalParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("complete withdrawal: %w", err)
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

	// Check sufficient reserved balance
	if player.ReservedBalance < params.Amount {
		return nil, domain.ErrValidation("insufficient reserved balance")
	}

	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxWithdrawalProcessed,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{ReservedBalance: -params.Amount},
		ExternalTransactionID: strPtr(extID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("complete withdrawal post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}
