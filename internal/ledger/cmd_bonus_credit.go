package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteBonusCredit credits the player's bonus balance.
func (e *Engine) ExecuteBonusCredit(ctx context.Context, tx pgx.Tx, params domain.BonusCreditParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("bonus credit: %w", err)
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

	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxBonusCredit,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{BonusBalance: params.Amount},
		ExternalTransactionID: strPtr(extID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("bonus credit post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}
