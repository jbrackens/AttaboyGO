package repository

import (
	"context"

	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX abstracts pgx.Tx and pgxpool.Pool so repositories work with both.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// PlayerRepository provides access to v2_players.
type PlayerRepository interface {
	// FindByID returns a player by ID.
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Player, error)

	// LockForUpdate acquires a row-level lock (SELECT FOR UPDATE) and returns the player.
	LockForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.Player, error)

	// Create inserts a new player.
	Create(ctx context.Context, db DBTX, player *domain.Player) error

	// UpdateBalances atomically updates balance columns using server-side arithmetic.
	// This uses raw pgx (Audit #1) with dynamic SET clauses.
	UpdateBalances(ctx context.Context, tx pgx.Tx, playerID uuid.UUID, delta domain.BalanceUpdate) (*domain.Player, error)
}

// TransactionRepository provides access to v2_transactions.
type TransactionRepository interface {
	// FindExisting checks the idempotency index for a duplicate transaction.
	FindExisting(ctx context.Context, db DBTX, key domain.IdempotencyKey) (*domain.Transaction, error)

	// Insert creates a new ledger entry with balance snapshot. Returns the inserted row.
	Insert(ctx context.Context, db DBTX, params domain.PostLedgerEntryParams, balances domain.Balances) (*domain.Transaction, error)

	// FindByID returns a transaction by ID.
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*domain.Transaction, error)

	// ListByPlayer returns transactions for a player, ordered by created_at DESC.
	// Supports cursor-based pagination.
	ListByPlayer(ctx context.Context, db DBTX, playerID uuid.UUID, cursor *string, limit int) ([]domain.Transaction, error)

	// ListByGameRound returns all transactions in a casino game round.
	ListByGameRound(ctx context.Context, db DBTX, gameRoundID string) ([]domain.Transaction, error)

	// DailySumByType returns the total amount of transactions of the given type
	// for a player since the start of the current calendar day (UTC).
	DailySumByType(ctx context.Context, db DBTX, playerID uuid.UUID, txType string) (int64, error)
}

// OutboxRepository provides access to the event_outbox table.
type OutboxRepository interface {
	// Insert writes an outbox event (within the same transaction as the ledger entry).
	Insert(ctx context.Context, db DBTX, draft domain.OutboxDraft) error

	// FetchUnpublished returns unpublished events for the outbox poller.
	FetchUnpublished(ctx context.Context, db DBTX, limit int) ([]domain.OutboxDraft, error)

	// MarkPublished deletes or marks events as published.
	MarkPublished(ctx context.Context, db DBTX, ids []int64) error
}

// AuthUserRepository provides access to auth_users.
type AuthUserRepository interface {
	// FindByEmail returns an auth user by email.
	FindByEmail(ctx context.Context, db DBTX, email string) (*domain.AuthUser, error)

	// Create inserts a new auth user.
	Create(ctx context.Context, db DBTX, user *domain.AuthUser) error
}

// ProfileRepository provides access to player_profiles.
type ProfileRepository interface {
	// FindByPlayerID returns a player profile.
	FindByPlayerID(ctx context.Context, db DBTX, playerID uuid.UUID) (*domain.PlayerProfile, error)

	// Create inserts a new player profile.
	Create(ctx context.Context, db DBTX, profile *domain.PlayerProfile) error

	// Update modifies a player profile.
	Update(ctx context.Context, db DBTX, profile *domain.PlayerProfile) error
}
