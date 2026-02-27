package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteTurnBonusToReal converts bonus balance to real balance.
// bonus_balance -= amount, balance += amount
func (e *Engine) ExecuteTurnBonusToReal(ctx context.Context, tx pgx.Tx, params domain.TurnBonusToRealParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("turn bonus: %w", err)
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

	// Check sufficient bonus balance
	if player.BonusBalance < params.Amount {
		return nil, domain.ErrValidation("insufficient bonus balance")
	}

	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxTurnBonusToReal,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{Balance: params.Amount, BonusBalance: -params.Amount},
		ExternalTransactionID: strPtr(extID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("turn bonus post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}
