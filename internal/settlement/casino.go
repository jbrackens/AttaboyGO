package settlement

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CasinoSettlement handles casino round settlement operations.
type CasinoSettlement struct {
	engine *ledger.Engine
	txRepo repository.TransactionRepository
}

// NewCasinoSettlement creates a casino settlement handler.
func NewCasinoSettlement(engine *ledger.Engine, txRepo repository.TransactionRepository) *CasinoSettlement {
	return &CasinoSettlement{engine: engine, txRepo: txRepo}
}

// GetRoundSummary aggregates all bets and wins in a casino round.
func (s *CasinoSettlement) GetRoundSummary(ctx context.Context, db repository.DBTX, gameRoundID string) (*domain.RoundSummary, error) {
	txs, err := s.txRepo.ListByGameRound(ctx, db, gameRoundID)
	if err != nil {
		return nil, fmt.Errorf("get round transactions: %w", err)
	}

	summary := &domain.RoundSummary{
		GameRoundID:  gameRoundID,
		Transactions: txs,
	}

	for _, tx := range txs {
		switch tx.Type {
		case domain.TxBet:
			summary.TotalBet += tx.Amount
			summary.BetCount++
		case domain.TxWin:
			summary.TotalWin += tx.Amount
			summary.WinCount++
		case domain.TxSettlementLoss:
			summary.IsSettled = true
		}
	}

	return summary, nil
}

// CreditCasinoWin credits a win within a casino round.
func (s *CasinoSettlement) CreditCasinoWin(ctx context.Context, tx pgx.Tx, params domain.CreditWinParams) (*domain.CommandResult, error) {
	return s.engine.ExecuteCreditWin(ctx, tx, params)
}

// SettleRoundLoss records a settlement_loss for an unsettled round.
// This releases any remaining round state (no balance change — bets already deducted).
func (s *CasinoSettlement) SettleRoundLoss(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, gameRoundID string) (*domain.CommandResult, error) {
	meta, _ := json.Marshal(map[string]string{"settlement": "round_loss"})
	roundID := gameRoundID
	entry, player, err := s.engine.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:      playerID,
		Type:          domain.TxSettlementLoss,
		Amount:        0,
		BalanceUpdate: domain.BalanceUpdate{},
		GameRoundID:   &roundID,
		Metadata:      meta,
	})
	if err != nil {
		return nil, fmt.Errorf("settle round loss: %w", err)
	}

	return &domain.CommandResult{
		Transaction: entry,
		Player:      player,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

// CancelCasinoRound bulk-cancels all bets and wins in a round.
func (s *CasinoSettlement) CancelCasinoRound(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, gameRoundID string) ([]domain.CommandResult, error) {
	txs, err := s.txRepo.ListByGameRound(ctx, tx, gameRoundID)
	if err != nil {
		return nil, fmt.Errorf("cancel round list: %w", err)
	}

	var results []domain.CommandResult
	for i, roundTx := range txs {
		if roundTx.Type != domain.TxBet && roundTx.Type != domain.TxWin {
			continue
		}
		result, err := s.engine.ExecuteCancelTransaction(ctx, tx, domain.CancelTransactionParams{
			PlayerID:              playerID,
			Amount:                roundTx.Amount,
			ExternalTransactionID: fmt.Sprintf("cancel-round-%s-%d", gameRoundID, i),
			TargetTransactionID:   roundTx.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cancel round tx %s: %w", roundTx.ID, err)
		}
		results = append(results, *result)
	}

	return results, nil
}
