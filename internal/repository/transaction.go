package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/infra"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type transactionRepo struct{}

// NewTransactionRepository returns a pgx-backed TransactionRepository.
func NewTransactionRepository() TransactionRepository {
	return &transactionRepo{}
}

func (r *transactionRepo) FindExisting(ctx context.Context, db DBTX, key domain.IdempotencyKey) (*domain.Transaction, error) {
	row := db.QueryRow(ctx, `
		SELECT id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
		       external_transaction_id, manufacturer_id, sub_transaction_id,
		       target_transaction_id, game_round_id, metadata, created_at
		FROM v2_transactions
		WHERE player_id = $1 AND manufacturer_id = $2
		  AND external_transaction_id = $3 AND sub_transaction_id = $4`,
		key.PlayerID, key.ManufacturerID, key.ExternalTransactionID, key.SubTransactionID)
	return scanTransaction(row)
}

func (r *transactionRepo) Insert(ctx context.Context, db DBTX, params domain.PostLedgerEntryParams, balances domain.Balances) (*domain.Transaction, error) {
	meta := params.Metadata
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}

	row := db.QueryRow(ctx, `
		INSERT INTO v2_transactions
		  (player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
		   external_transaction_id, manufacturer_id, sub_transaction_id,
		   target_transaction_id, game_round_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
		          external_transaction_id, manufacturer_id, sub_transaction_id,
		          target_transaction_id, game_round_id, metadata, created_at`,
		params.PlayerID,
		string(params.Type),
		infra.Int64ToNumeric(params.Amount),
		infra.Int64ToNumeric(balances.Balance),
		infra.Int64ToNumeric(balances.BonusBalance),
		infra.Int64ToNumeric(balances.ReservedBalance),
		params.ExternalTransactionID,
		params.ManufacturerID,
		params.SubTransactionID,
		params.TargetTransactionID,
		params.GameRoundID,
		meta,
	)
	return scanTransaction(row)
}

func (r *transactionRepo) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Transaction, error) {
	row := db.QueryRow(ctx, `
		SELECT id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
		       external_transaction_id, manufacturer_id, sub_transaction_id,
		       target_transaction_id, game_round_id, metadata, created_at
		FROM v2_transactions WHERE id = $1`, id)
	return scanTransaction(row)
}

func (r *transactionRepo) ListByPlayer(ctx context.Context, db DBTX, playerID uuid.UUID, cursor *string, limit int) ([]domain.Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows pgx.Rows
	var err error
	if cursor != nil {
		rows, err = db.Query(ctx, `
			SELECT id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
			       external_transaction_id, manufacturer_id, sub_transaction_id,
			       target_transaction_id, game_round_id, metadata, created_at
			FROM v2_transactions
			WHERE player_id = $1
			  AND (created_at, id) <= ((SELECT created_at, id FROM v2_transactions WHERE id = $2))
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, playerID, *cursor, limit)
	} else {
		rows, err = db.Query(ctx, `
			SELECT id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
			       external_transaction_id, manufacturer_id, sub_transaction_id,
			       target_transaction_id, game_round_id, metadata, created_at
			FROM v2_transactions
			WHERE player_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, playerID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	return collectTransactions(rows)
}

func (r *transactionRepo) ListByGameRound(ctx context.Context, db DBTX, gameRoundID string) ([]domain.Transaction, error) {
	rows, err := db.Query(ctx, `
		SELECT id, player_id, type, amount, balance_after, bonus_balance_after, reserved_balance_after,
		       external_transaction_id, manufacturer_id, sub_transaction_id,
		       target_transaction_id, game_round_id, metadata, created_at
		FROM v2_transactions
		WHERE game_round_id = $1
		ORDER BY created_at ASC`, gameRoundID)
	if err != nil {
		return nil, fmt.Errorf("query round transactions: %w", err)
	}
	defer rows.Close()

	return collectTransactions(rows)
}

func scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var tx domain.Transaction
	var amountNum, balNum, bonusNum, reservedNum pgtype.Numeric
	err := row.Scan(
		&tx.ID, &tx.PlayerID, &tx.Type,
		&amountNum, &balNum, &bonusNum, &reservedNum,
		&tx.ExternalTransactionID, &tx.ManufacturerID, &tx.SubTransactionID,
		&tx.TargetTransactionID, &tx.GameRoundID, &tx.Metadata, &tx.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan transaction: %w", err)
	}

	var convErr error
	tx.Amount, convErr = infra.NumericToInt64(amountNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert amount: %w", convErr)
	}
	tx.BalanceAfter, convErr = infra.NumericToInt64(balNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert balance_after: %w", convErr)
	}
	tx.BonusBalanceAfter, convErr = infra.NumericToInt64(bonusNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert bonus_balance_after: %w", convErr)
	}
	tx.ReservedBalanceAfter, convErr = infra.NumericToInt64(reservedNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert reserved_balance_after: %w", convErr)
	}

	return &tx, nil
}

func collectTransactions(rows pgx.Rows) ([]domain.Transaction, error) {
	var txs []domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		var amountNum, balNum, bonusNum, reservedNum pgtype.Numeric
		err := rows.Scan(
			&tx.ID, &tx.PlayerID, &tx.Type,
			&amountNum, &balNum, &bonusNum, &reservedNum,
			&tx.ExternalTransactionID, &tx.ManufacturerID, &tx.SubTransactionID,
			&tx.TargetTransactionID, &tx.GameRoundID, &tx.Metadata, &tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction row: %w", err)
		}
		var convErr error
		tx.Amount, convErr = infra.NumericToInt64(amountNum)
		if convErr != nil {
			return nil, convErr
		}
		tx.BalanceAfter, convErr = infra.NumericToInt64(balNum)
		if convErr != nil {
			return nil, convErr
		}
		tx.BonusBalanceAfter, convErr = infra.NumericToInt64(bonusNum)
		if convErr != nil {
			return nil, convErr
		}
		tx.ReservedBalanceAfter, convErr = infra.NumericToInt64(reservedNum)
		if convErr != nil {
			return nil, convErr
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}
