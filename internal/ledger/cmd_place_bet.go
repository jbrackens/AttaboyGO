package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecutePlaceBet deducts from the player's balance (real-first, then bonus).
// Tracks the real/bonus split in metadata for matching on win.
func (e *Engine) ExecutePlaceBet(ctx context.Context, tx pgx.Tx, params domain.PlaceBetParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("place bet: %w", err)
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

	// Bet split: real balance first, then bonus
	totalAvailable := player.Balance + player.BonusBalance
	if totalAvailable < params.Amount {
		return nil, domain.ErrInsufficientBalance()
	}

	realBet := params.Amount
	var bonusBet int64
	if realBet > player.Balance {
		realBet = player.Balance
		bonusBet = params.Amount - realBet
	}

	// Build metadata with bet split tracking
	meta := mergeMeta(params.Metadata, map[string]interface{}{
		"realBet":  realBet,
		"bonusBet": bonusBet,
	})

	roundID := params.GameRoundID
	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxBet,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{Balance: -realBet, BonusBalance: -bonusBet},
		ExternalTransactionID: strPtr(extID),
		ManufacturerID:        strPtr(mfgID),
		SubTransactionID:      strPtr(subID),
		GameRoundID:           strPtr(roundID),
		Metadata:              meta,
	})
	if err != nil {
		return nil, fmt.Errorf("place bet post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

func mergeMeta(base json.RawMessage, extra map[string]interface{}) json.RawMessage {
	merged := make(map[string]interface{})
	if base != nil && len(base) > 0 {
		_ = json.Unmarshal(base, &merged)
	}
	for k, v := range extra {
		merged[k] = v
	}
	out, _ := json.Marshal(merged)
	return out
}
