package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteDeposit credits the player's real balance.
// Pattern: Lock → Idempotency → PostLedgerEntry
func (e *Engine) ExecuteDeposit(ctx context.Context, tx pgx.Tx, params domain.DepositParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("deposit: %w", err)
	}

	// Idempotency check
	extID := params.ExternalTransactionID
	mfgID := params.ManufacturerID
	subID := params.SubTransactionID
	if extID != "" {
		existing, err := e.FindExistingTransaction(ctx, tx, domain.IdempotencyKey{
			PlayerID:              params.PlayerID,
			ManufacturerID:        mfgID,
			ExternalTransactionID: extID,
			SubTransactionID:      subID,
		})
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &domain.CommandResult{Transaction: existing, Player: player, Idempotent: true}, nil
		}
	}

	// Post ledger entry: balance += amount
	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxDeposit,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{Balance: params.Amount},
		ExternalTransactionID: strPtr(extID),
		ManufacturerID:        strPtr(mfgID),
		SubTransactionID:      strPtr(subID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("deposit post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ensureJSON(data json.RawMessage) json.RawMessage {
	if data == nil {
		return json.RawMessage(`{}`)
	}
	return data
}
