package settlement

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PredictionSettlement handles prediction market settlement with oracle attestation.
type PredictionSettlement struct {
	engine *ledger.Engine
}

// NewPredictionSettlement creates a prediction settlement handler.
func NewPredictionSettlement(engine *ledger.Engine) *PredictionSettlement {
	return &PredictionSettlement{engine: engine}
}

// ValidateAttestation verifies the oracle attestation before settlement.
func (s *PredictionSettlement) ValidateAttestation(attestation domain.Attestation) error {
	return domain.ValidateAttestation(attestation)
}

// PlaceStake deducts a stake from the player's balance.
func (s *PredictionSettlement) PlaceStake(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, marketID uuid.UUID, outcome string, amount int64) (*domain.CommandResult, error) {
	meta, _ := json.Marshal(map[string]interface{}{
		"marketId": marketID.String(),
		"outcome":  outcome,
		"type":     "prediction_stake",
	})
	return s.engine.ExecutePlaceBet(ctx, tx, domain.PlaceBetParams{
		PlayerID:              playerID,
		Amount:                amount,
		ExternalTransactionID: fmt.Sprintf("pred-stake-%s-%s", marketID, outcome),
		GameRoundID:           marketID.String(),
		Metadata:              meta,
	})
}

// SettleOutcomeWin credits a winning prediction. Requires valid attestation.
func (s *PredictionSettlement) SettleOutcomeWin(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, marketID uuid.UUID, winAmount int64, attestation domain.Attestation) (*domain.CommandResult, error) {
	if err := s.ValidateAttestation(attestation); err != nil {
		return nil, fmt.Errorf("prediction settlement: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"marketId":    marketID.String(),
		"settlement":  "prediction_win",
		"attestation": attestation,
	})
	return s.engine.ExecuteCreditWin(ctx, tx, domain.CreditWinParams{
		PlayerID:              playerID,
		Amount:                winAmount,
		ExternalTransactionID: fmt.Sprintf("pred-win-%s", marketID),
		GameRoundID:           marketID.String(),
		Metadata:              meta,
	})
}

// SettleOutcomeLoss records a losing prediction (no balance change).
func (s *PredictionSettlement) SettleOutcomeLoss(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, marketID uuid.UUID, attestation domain.Attestation) (*domain.CommandResult, error) {
	if err := s.ValidateAttestation(attestation); err != nil {
		return nil, fmt.Errorf("prediction loss settlement: %w", err)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"marketId":    marketID.String(),
		"settlement":  "prediction_loss",
		"attestation": attestation,
	})
	entry, player, err := s.engine.PostLedgerEntry(ctx, tx, domain.PostLedgerEntryParams{
		PlayerID:              playerID,
		Type:                  domain.TxSettlementLoss,
		Amount:                0,
		BalanceUpdate:         domain.BalanceUpdate{},
		ExternalTransactionID: strPtr(fmt.Sprintf("pred-loss-%s", marketID)),
		GameRoundID:           strPtr(marketID.String()),
		Metadata:              meta,
	})
	if err != nil {
		return nil, fmt.Errorf("settle prediction loss: %w", err)
	}
	return &domain.CommandResult{
		Transaction: entry,
		Player:      player,
		Events:      []domain.OutboxDraft{domain.NewTransactionPostedEvent(entry)},
	}, nil
}

// VoidMarket cancels all stakes in a voided market (returns stakes to players).
func (s *PredictionSettlement) VoidMarket(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, stakeTxID uuid.UUID, stakeAmount int64) (*domain.CommandResult, error) {
	return s.engine.ExecuteCancelTransaction(ctx, tx, domain.CancelTransactionParams{
		PlayerID:              playerID,
		Amount:                stakeAmount,
		ExternalTransactionID: fmt.Sprintf("pred-void-%s", stakeTxID),
		TargetTransactionID:   stakeTxID,
	})
}
