package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/infra"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type playerRepo struct{}

// NewPlayerRepository returns a pgx-backed PlayerRepository.
func NewPlayerRepository() PlayerRepository {
	return &playerRepo{}
}

func (r *playerRepo) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Player, error) {
	row := db.QueryRow(ctx, `
		SELECT id, balance, bonus_balance, reserved_balance, currency, created_at, updated_at
		FROM v2_players WHERE id = $1`, id)
	return scanPlayer(row)
}

func (r *playerRepo) LockForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.Player, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, balance, bonus_balance, reserved_balance, currency, created_at, updated_at
		FROM v2_players WHERE id = $1 FOR UPDATE`, id)
	return scanPlayer(row)
}

func (r *playerRepo) Create(ctx context.Context, db DBTX, player *domain.Player) error {
	_, err := db.Exec(ctx, `
		INSERT INTO v2_players (id, balance, bonus_balance, reserved_balance, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		player.ID,
		infra.Int64ToNumeric(player.Balance),
		infra.Int64ToNumeric(player.BonusBalance),
		infra.Int64ToNumeric(player.ReservedBalance),
		player.Currency,
		player.CreatedAt,
		player.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert player: %w", err)
	}
	return nil
}

// UpdateBalances uses server-side arithmetic with dynamic SET clauses (Audit #1).
func (r *playerRepo) UpdateBalances(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, delta domain.BalanceUpdate) (*domain.Player, error) {
	setClauses := []string{"updated_at = now()"}
	args := []interface{}{}
	argIdx := 1

	if delta.HasBalanceDelta() {
		setClauses = append(setClauses, fmt.Sprintf("balance = balance + $%d", argIdx))
		args = append(args, infra.Int64ToNumeric(delta.Balance))
		argIdx++
	}
	if delta.HasBonusDelta() {
		setClauses = append(setClauses, fmt.Sprintf("bonus_balance = bonus_balance + $%d", argIdx))
		args = append(args, infra.Int64ToNumeric(delta.BonusBalance))
		argIdx++
	}
	if delta.HasReservedDelta() {
		setClauses = append(setClauses, fmt.Sprintf("reserved_balance = reserved_balance + $%d", argIdx))
		args = append(args, infra.Int64ToNumeric(delta.ReservedBalance))
		argIdx++
	}

	args = append(args, playerID)
	query := fmt.Sprintf(`
		UPDATE v2_players SET %s
		WHERE id = $%d
		RETURNING id, balance, bonus_balance, reserved_balance, currency, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

	row := tx.QueryRow(ctx, query, args...)
	return scanPlayer(row)
}

func scanPlayer(row pgx.Row) (*domain.Player, error) {
	var p domain.Player
	var balNum, bonusNum, reservedNum pgtype.Numeric
	err := row.Scan(&p.ID, &balNum, &bonusNum, &reservedNum, &p.Currency, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan player: %w", err)
	}

	var convErr error
	p.Balance, convErr = infra.NumericToInt64(balNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert balance: %w", convErr)
	}
	p.BonusBalance, convErr = infra.NumericToInt64(bonusNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert bonus_balance: %w", convErr)
	}
	p.ReservedBalance, convErr = infra.NumericToInt64(reservedNum)
	if convErr != nil {
		return nil, fmt.Errorf("convert reserved_balance: %w", convErr)
	}

	return &p, nil
}
