package ledger

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Engine provides the 3 foundational ledger operations:
//   1. LockPlayerForUpdate — row-level pessimistic lock
//   2. FindExistingTransaction — idempotency check
//   3. PostLedgerEntry — atomic balance update + append-only insert + outbox event
type Engine struct {
	players      repository.PlayerRepository
	transactions repository.TransactionRepository
	outbox       repository.OutboxRepository
}

// NewEngine creates a ledger engine with the given repositories.
func NewEngine(
	players repository.PlayerRepository,
	transactions repository.TransactionRepository,
	outbox repository.OutboxRepository,
) *Engine {
	return &Engine{
		players:      players,
		transactions: transactions,
		outbox:       outbox,
	}
}

// LockPlayerForUpdate acquires a row-level lock and returns the player.
// Must be called within a transaction.
func (e *Engine) LockPlayerForUpdate(ctx context.Context, tx pgx.Tx, playerID uuid.UUID) (*domain.Player, error) {
	player, err := e.players.LockForUpdate(ctx, tx, playerID)
	if err != nil {
		return nil, fmt.Errorf("lock player: %w", err)
	}
	if player == nil {
		return nil, domain.ErrNotFound("player", playerID.String())
	}
	return player, nil
}

// FindExistingTransaction checks if a transaction with the same idempotency key exists.
// Returns nil if no duplicate found.
func (e *Engine) FindExistingTransaction(ctx context.Context, tx pgx.Tx, key domain.IdempotencyKey) (*domain.Transaction, error) {
	existing, err := e.transactions.FindExisting(ctx, tx, key)
	if err != nil {
		return nil, fmt.Errorf("find existing transaction: %w", err)
	}
	return existing, nil
}

// PostLedgerEntry atomically updates player balances and inserts a ledger entry.
// This is the core write primitive — all 9 commands delegate to this.
//
// Steps:
//  1. Update player balances using server-side arithmetic (dynamic SET clauses)
//  2. Insert transaction with the post-update balance snapshot
//  3. Insert outbox event
//
// All 3 steps run within the caller's transaction.
func (e *Engine) PostLedgerEntry(ctx context.Context, tx pgx.Tx, params domain.PostLedgerEntryParams) (*domain.Transaction, *domain.Player, error) {
	// Step 1: Atomic balance update with server-side arithmetic
	updatedPlayer, err := e.players.UpdateBalances(ctx, tx, params.PlayerID, params.BalanceUpdate)
	if err != nil {
		return nil, nil, fmt.Errorf("update balances: %w", err)
	}

	// Step 2: Insert ledger entry with post-update balance snapshot
	entry, err := e.transactions.Insert(ctx, tx, params, updatedPlayer.Balances)
	if err != nil {
		return nil, nil, fmt.Errorf("insert transaction: %w", err)
	}

	// Step 3: Insert outbox event (same transaction for atomicity)
	event := domain.NewTransactionPostedEvent(entry)
	if err := e.outbox.Insert(ctx, tx, event); err != nil {
		return nil, nil, fmt.Errorf("insert outbox event: %w", err)
	}

	return entry, updatedPlayer, nil
}
