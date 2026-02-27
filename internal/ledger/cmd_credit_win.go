package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/jackc/pgx/v5"
)

// ExecuteCreditWin credits a win back to the player.
// Win split follows the bet split: proportional allocation between real and bonus.
//
// Win-split algorithm (from Node.js):
//   - If player has active bonus balance → all win goes to bonus
//   - Otherwise → proportional split based on original bet real/bonus ratio
func (e *Engine) ExecuteCreditWin(ctx context.Context, tx pgx.Tx, params domain.CreditWinParams) (*domain.CommandResult, error) {
	if err := domain.ValidatePositiveAmount(params.Amount); err != nil {
		return nil, err
	}

	// Lock
	player, err := e.LockPlayerForUpdate(ctx, tx, params.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("credit win: %w", err)
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

	// Compute win split based on bet history in this round
	realWin, bonusWin := computeWinSplit(ctx, e.transactions, tx, player, params)

	winType := params.WinType
	if winType == "" {
		winType = domain.CasinoWinNormal
	}

	meta := mergeMeta(params.Metadata, map[string]interface{}{
		"realWin":  realWin,
		"bonusWin": bonusWin,
		"winType":  string(winType),
	})

	roundID := params.GameRoundID
	entry, updatedPlayer, err := e.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              params.PlayerID,
		Type:                  domain.TxWin,
		Amount:                params.Amount,
		BalanceUpdate:         domain.BalanceUpdate{Balance: realWin, BonusBalance: bonusWin},
		ExternalTransactionID: strPtr(extID),
		ManufacturerID:        strPtr(mfgID),
		SubTransactionID:      strPtr(subID),
		GameRoundID:           strPtr(roundID),
		Metadata:              meta,
	})
	if err != nil {
		return nil, fmt.Errorf("credit win post: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      updatedPlayer,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

// computeWinSplit determines how to split a win between real and bonus balance.
// V1 invariant: if bonus is active → all to bonus; else proportional.
func computeWinSplit(ctx context.Context, txRepo repository.TransactionRepository, tx pgx.Tx, player *domain.Player, params domain.CreditWinParams) (realWin, bonusWin int64) {
	// If player has active bonus balance, all win goes to bonus
	if player.BonusBalance > 0 {
		return 0, params.Amount
	}

	// Look up bet history in this round to determine proportion
	if params.GameRoundID != "" {
		bets, err := txRepo.ListByGameRound(ctx, tx, params.GameRoundID)
		if err == nil && len(bets) > 0 {
			var totalRealBet, totalBonusBet int64
			for _, bet := range bets {
				if bet.Type == domain.TxBet {
					var meta map[string]interface{}
					if err := json.Unmarshal(bet.Metadata, &meta); err == nil {
						if rb, ok := meta["realBet"].(float64); ok {
							totalRealBet += int64(rb)
						}
						if bb, ok := meta["bonusBet"].(float64); ok {
							totalBonusBet += int64(bb)
						}
					}
				}
			}

			totalBet := totalRealBet + totalBonusBet
			if totalBet > 0 && totalBonusBet > 0 {
				// Proportional split
				bonusWin = (params.Amount * totalBonusBet) / totalBet
				realWin = params.Amount - bonusWin
				return realWin, bonusWin
			}
		}
	}

	// Default: all to real balance
	return params.Amount, 0
}
