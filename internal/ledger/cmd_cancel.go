package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ExecuteCancelTransaction reverses a previous transaction.
// The cancellation type is derived from the original transaction type.
func (e *Engine) ExecuteCancelTransaction(ctx context.Context, tx pgx.Tx, params domain.CancelTransactionParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("cancel: %w", err)
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

	// Find the target transaction to determine cancellation type and reversal amounts
	target, err := e.transactions.FindByID(ctx, tx, params.TargetTransactionID)
	if err != nil {
		return nil, fmt.Errorf("cancel find target: %w", err)
	}
	if target == nil {
		return nil, domain.ErrNotFound("transaction", params.TargetTransactionID.String())
	}

	cancelType, ok := domain.CancellationTypeMap[target.Type]
	if !ok {
		return nil, domain.ErrValidation(fmt.Sprintf("cannot cancel transaction type: %s", target.Type))
	}

	// Reverse the balance changes from the original transaction
	var delta domain.BalanceUpdate
	switch target.Type {
	case domain.TxDeposit:
		delta = domain.BalanceUpdate{Balance: -params.Amount}
	case domain.TxBet:
		// Restore the real/bonus split from the original bet metadata
		realBet, bonusBet := extractBetSplit(target)
		delta = domain.BalanceUpdate{Balance: realBet, BonusBalance: bonusBet}
	case domain.TxWin:
		// Reverse the real/bonus split from the original win metadata
		realWin, bonusWin := extractWinSplit(target)
		delta = domain.BalanceUpdate{Balance: -realWin, BonusBalance: -bonusWin}
	case domain.TxWithdrawal:
		// Restore from reserved back to balance
		delta = domain.BalanceUpdate{Balance: params.Amount, ReservedBalance: -params.Amount}
	}

	targetID := params.TargetTransactionID
	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  cancelType,
		Amount:                params.Amount,
		BalanceUpdate:         delta,
		ExternalTransactionID: strPtr(extID),
		ManufacturerID:        strPtr(mfgID),
		SubTransactionID:      strPtr(subID),
		TargetTransactionID:   &targetID,
		Metadata:              ensureJSON(params.Metadata),
	})
	if err != nil {
		return nil, fmt.Errorf("cancel post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

func extractBetSplit(tx *domain.Transaction) (realBet, bonusBet int64) {
	var meta map[string]interface{}
	if err := jsonUnmarshal(tx.Metadata, &meta); err != nil {
		return tx.Amount, 0
	}
	if rb, ok := meta["realBet"].(float64); ok {
		realBet = int64(rb)
	}
	if bb, ok := meta["bonusBet"].(float64); ok {
		bonusBet = int64(bb)
	}
	if realBet+bonusBet == 0 {
		return tx.Amount, 0
	}
	return realBet, bonusBet
}

func extractWinSplit(tx *domain.Transaction) (realWin, bonusWin int64) {
	var meta map[string]interface{}
	if err := jsonUnmarshal(tx.Metadata, &meta); err != nil {
		return tx.Amount, 0
	}
	if rw, ok := meta["realWin"].(float64); ok {
		realWin = int64(rw)
	}
	if bw, ok := meta["bonusWin"].(float64); ok {
		bonusWin = int64(bw)
	}
	if realWin+bonusWin == 0 {
		return tx.Amount, 0
	}
	return realWin, bonusWin
}
