package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteForfeitBonus removes bonus balance (forfeit or loss).
// bonus_balance -= amount
// IsBonusLost distinguishes between voluntary forfeit and admin-initiated loss.
func (e *Engine) ExecuteForfeitBonus(ctx context.Context, tx pgx.Tx, params domain.ForfeitBonusParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("forfeit bonus: %w", err)
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
		return nil, domain.ErrValidation("insufficient bonus balance for forfeit")
	}

	txType := domain.TxBonusForfeit
	if params.IsBonusLost {
		txType = domain.TxBonusLost
	}

	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  txType,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{BonusBalance: -params.Amount},
		ExternalTransactionID: strPtr(extID),
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("forfeit bonus post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

// jsonUnmarshal is a helper that wraps json.Unmarshal.
func jsonUnmarshal(data json.RawMessage, v interface{}) error {
	if data == nil {
		return fmt.Errorf("nil json data")
	}
	return json.Unmarshal(data, v)
}
