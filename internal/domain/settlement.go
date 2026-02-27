package domain

import (
	"time"

	"github.com/google/uuid"
)

// --- Casino Settlement ---

// RoundSummary aggregates all bets and wins in a casino round.
type RoundSummary struct {
	GameRoundID  string
	TotalBet     int64
	TotalWin     int64
	BetCount     int
	WinCount     int
	IsSettled    bool
	Transactions []Transaction
}

// --- Sportsbook ---

// BetStatus tracks the lifecycle of a sportsbook bet.
type BetStatus string

const (
	BetStatusPending  BetStatus = "pending"
	BetStatusWon      BetStatus = "won"
	BetStatusLost     BetStatus = "lost"
	BetStatusVoid     BetStatus = "void"
	BetStatusCashout  BetStatus = "cashout"
)

// SportsbookBet represents a placed bet.
type SportsbookBet struct {
	ID           uuid.UUID  `json:"id"`
	PlayerID     uuid.UUID  `json:"player_id"`
	Stake        int64      `json:"stake"`
	Odds         float64    `json:"odds"`
	Status       BetStatus  `json:"status"`
	IsParlay     bool       `json:"is_parlay"`
	Selections   []BetSelection `json:"selections,omitempty"`
	TransactionID uuid.UUID `json:"transaction_id"`
	SettledAt    *time.Time `json:"settled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// BetSelection represents one leg of a bet (or the single selection for singles).
type BetSelection struct {
	ID          uuid.UUID `json:"id"`
	BetID       uuid.UUID `json:"bet_id"`
	EventID     uuid.UUID `json:"event_id"`
	MarketID    uuid.UUID `json:"market_id"`
	SelectionID uuid.UUID `json:"selection_id"`
	Odds        float64   `json:"odds"`
	Result      string    `json:"result,omitempty"`
}

// --- Prediction Markets ---

// Attestation is the oracle proof required for prediction settlement.
type Attestation struct {
	Provider      string `json:"provider"`
	AttestationID string `json:"attestation_id"`
	Digest        string `json:"digest"` // hex 32-128 chars
	IssuedAt      time.Time `json:"issued_at"`
}

// PredictionMarket represents a prediction market.
type PredictionMarket struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"` // open, closed, settled, voided
	OutcomeA    string    `json:"outcome_a"`
	OutcomeB    string    `json:"outcome_b"`
	PoolA       int64     `json:"pool_a"`
	PoolB       int64     `json:"pool_b"`
	SettledAt   *time.Time `json:"settled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PredictionStake represents a player's position in a prediction market.
type PredictionStake struct {
	ID        uuid.UUID `json:"id"`
	PlayerID  uuid.UUID `json:"player_id"`
	MarketID  uuid.UUID `json:"market_id"`
	Outcome   string    `json:"outcome"` // "a" or "b"
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
