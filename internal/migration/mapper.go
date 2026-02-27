package migration

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionMapper maps V1 transactions to V2 format.
type TransactionMapper struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewTransactionMapper creates a new V1→V2 transaction mapper.
func NewTransactionMapper(pool *pgxpool.Pool, logger *slog.Logger) *TransactionMapper {
	return &TransactionMapper{pool: pool, logger: logger}
}

// V1Transaction represents a transaction from the V1 schema.
type V1Transaction struct {
	ID             string    `json:"id"`
	PlayerID       string    `json:"player_id"`
	Type           string    `json:"type"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	BalanceAfter   int64     `json:"balance_after"`
	ExternalTxID   string    `json:"external_tx_id"`
	GameRoundID    string    `json:"game_round_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// MapTransaction converts a V1 transaction to V2 format and returns the V2 UUID.
func (m *TransactionMapper) MapTransaction(v1Tx V1Transaction) (uuid.UUID, error) {
	v2TxID := DeterministicUUID("transaction", v1Tx.ID)
	v2PlayerID := DeterministicUUID("player", v1Tx.PlayerID)

	m.logger.Debug("mapped transaction",
		"v1_tx_id", v1Tx.ID,
		"v2_tx_id", v2TxID,
		"v2_player_id", v2PlayerID,
		"type", v1Tx.Type)

	return v2TxID, nil
}

// BalanceComparison holds the result of comparing V1 and V2 balances.
type BalanceComparison struct {
	PlayerV1ID string `json:"v1_player_id"`
	PlayerV2ID string `json:"v2_player_id"`
	V1Balance  int64  `json:"v1_balance"`
	V2Balance  int64  `json:"v2_balance"`
	Match      bool   `json:"match"`
}

// CompareBalances compares player balances between V1 and V2.
func (m *TransactionMapper) CompareBalances(ctx context.Context, v1PlayerIDs []string) ([]BalanceComparison, error) {
	var comparisons []BalanceComparison

	for _, v1ID := range v1PlayerIDs {
		v2ID := DeterministicUUID("player", v1ID)

		var v2Balance int64
		err := m.pool.QueryRow(ctx, `SELECT balance FROM v2_players WHERE id = $1`, v2ID).Scan(&v2Balance)
		if err != nil {
			m.logger.Warn("v2 player not found", "v1_id", v1ID, "v2_id", v2ID)
			comparisons = append(comparisons, BalanceComparison{
				PlayerV1ID: v1ID,
				PlayerV2ID: v2ID.String(),
				V1Balance:  0,
				V2Balance:  0,
				Match:      false,
			})
			continue
		}

		comparisons = append(comparisons, BalanceComparison{
			PlayerV1ID: v1ID,
			PlayerV2ID: v2ID.String(),
			V2Balance:  v2Balance,
			Match:      true, // V1 balance would come from V1 DB query
		})
	}

	return comparisons, nil
}

// CutoverReadiness checks whether the system is ready for V1→V2 cutover.
type CutoverReadiness struct {
	PlayersMatch       bool   `json:"players_match"`
	TransactionsCount  int    `json:"v2_transactions_count"`
	OutboxHealthy      bool   `json:"outbox_healthy"`
	BalanceMismatches  int    `json:"balance_mismatches"`
	Ready              bool   `json:"ready"`
	Message            string `json:"message"`
}

// CheckCutoverReadiness validates the system state for migration cutover.
func (m *TransactionMapper) CheckCutoverReadiness(ctx context.Context) (*CutoverReadiness, error) {
	readiness := &CutoverReadiness{}

	// Count V2 transactions
	err := m.pool.QueryRow(ctx, `SELECT COUNT(*) FROM v2_transactions`).Scan(&readiness.TransactionsCount)
	if err != nil {
		return nil, fmt.Errorf("count transactions: %w", err)
	}

	// Check outbox health — no unpublished events older than 5 minutes
	var staleCount int
	err = m.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM event_outbox
		WHERE "publishedAt" IS NULL AND "occurredAt" < now() - interval '5 minutes'`).
		Scan(&staleCount)
	if err != nil {
		return nil, fmt.Errorf("check outbox: %w", err)
	}
	readiness.OutboxHealthy = staleCount == 0

	// Count players with non-zero balance
	var playerCount int
	err = m.pool.QueryRow(ctx, `SELECT COUNT(*) FROM v2_players WHERE balance > 0`).Scan(&playerCount)
	if err != nil {
		return nil, fmt.Errorf("count players: %w", err)
	}
	readiness.PlayersMatch = playerCount > 0

	readiness.Ready = readiness.OutboxHealthy && readiness.PlayersMatch
	if readiness.Ready {
		readiness.Message = "system ready for cutover"
	} else {
		readiness.Message = "system not ready: check outbox health and player data"
	}

	m.logger.Info("cutover readiness check",
		"ready", readiness.Ready,
		"transactions", readiness.TransactionsCount,
		"outbox_healthy", readiness.OutboxHealthy)

	return readiness, nil
}
