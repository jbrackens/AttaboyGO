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

// SportsbookSettlement handles sportsbook bet settlement.
type SportsbookSettlement struct {
	engine *ledger.Engine
	txRepo repository.TransactionRepository
}

// NewSportsbookSettlement creates a sportsbook settlement handler.
func NewSportsbookSettlement(engine *ledger.Engine, txRepo repository.TransactionRepository) *SportsbookSettlement {
	return &SportsbookSettlement{engine: engine, txRepo: txRepo}
}

// FindBetTransaction locates the original bet transaction by ID.
func (s *SportsbookSettlement) FindBetTransaction(ctx context.Context, db repository.DBTX, betTxID uuid.UUID) (*domain.Transaction, error) {
	tx, err := s.txRepo.FindByID(ctx, db, betTxID)
	if err != nil {
		return nil, fmt.Errorf("find bet tx: %w", err)
	}
	if tx == nil {
		return nil, domain.ErrNotFound("transaction", betTxID.String())
	}
	if tx.Type != domain.TxBet {
		return nil, domain.ErrValidation(fmt.Sprintf("transaction %s is not a bet (type: %s)", betTxID, tx.Type))
	}
	return tx, nil
}

// SettleBetWin credits a winning bet.
func (s *SportsbookSettlement) SettleBetWin(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, betTxID uuid.UUID, winAmount int64) (*domain.CommandResult, error) {
	meta, _ := json.Marshal(map[string]interface{}{
		"settlement": "bet_win",
		"betTxId":    betTxID.String(),
	})
	return s.engine.ExecuteCreditWin(ctx, tx, domain.CreditWinParams{
		PlayerID:              playerID,
		Amount:                winAmount,
		ExternalTransactionID: fmt.Sprintf("settle-win-%s", betTxID),
		GameRoundID:           betTxID.String(),
		WinType:               domain.CasinoWinNormal,
		Metadata:              meta,
	})
}

// SettleBetLoss records a loss (no balance change — bet already deducted).
func (s *SportsbookSettlement) SettleBetLoss(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, betTxID uuid.UUID) (*domain.CommandResult, error) {
	meta, _ := json.Marshal(map[string]interface{}{
		"settlement": "bet_loss",
		"betTxId":    betTxID.String(),
	})
	entry, player, err := s.engine.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              playerID,
		Type:                  domain.TxSettlementLoss,
		Amount:                0,
		BalanceUpdate:         domain.BalanceUpdate{},
		ExternalTransactionID: strPtr(fmt.Sprintf("settle-loss-%s", betTxID)),
		Metadata:              meta,
	})
	if err != nil {
		return nil, fmt.Errorf("settle bet loss: %w", err)
	}
	return &domain.CommandResult{
		Transaction: entry,
		Player:      player,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

// SettleBetVoid cancels a bet (returns stake to player).
func (s *SportsbookSettlement) SettleBetVoid(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, betTx *domain.Transaction) (*domain.CommandResult, error) {
	return s.engine.ExecuteCancelTransaction(ctx, tx, domain.CancelTransactionParams{
		PlayerID:              playerID,
		Amount:                betTx.Amount,
		ExternalTransactionID: fmt.Sprintf("settle-void-%s", betTx.ID),
		TargetTransactionID:   betTx.ID,
	})
}

// RollbackSettlement reverses a previous settlement (win or loss).
func (s *SportsbookSettlement) RollbackSettlement(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, settlementTxID uuid.UUID) (*domain.CommandResult, error) {
	settleTx, err := s.txRepo.FindByID(ctx, tx, settlementTxID)
	if err != nil {
		return nil, fmt.Errorf("rollback find settlement: %w", err)
	}
	if settleTx == nil {
		return nil, domain.ErrNotFound("settlement transaction", settlementTxID.String())
	}

	return s.engine.ExecuteCancelTransaction(ctx, tx, domain.CancelTransactionParams{
		PlayerID:              playerID,
		Amount:                settleTx.Amount,
		ExternalTransactionID: fmt.Sprintf("rollback-%s", settlementTxID),
		TargetTransactionID:   settlementTxID,
	})
}

func strPtr(s string) *string { return &s }
