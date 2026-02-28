package domain

import (
	"time"

	"github.com/google/uuid"
)

// Sport represents a sports category.
type Sport struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	SortOrder int       `json:"sort_order"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// SportsEvent represents a sporting event.
type SportsEvent struct {
	ID        uuid.UUID `json:"id"`
	SportID   uuid.UUID `json:"sport_id"`
	League    *string   `json:"league,omitempty"`
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
	ScoreHome int       `json:"score_home"`
	ScoreAway int       `json:"score_away"`
	CreatedAt time.Time `json:"created_at"`
}

// SportsMarket represents a betting market within an event.
type SportsMarket struct {
	ID         uuid.UUID `json:"id"`
	EventID    uuid.UUID `json:"event_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Specifiers *string   `json:"specifiers,omitempty"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

// SportsSelection represents a selection within a market.
type SportsSelection struct {
	ID              uuid.UUID `json:"id"`
	MarketID        uuid.UUID `json:"market_id"`
	Name            string    `json:"name"`
	OddsDecimal     int       `json:"odds_decimal"`
	OddsFractional  *string   `json:"odds_fractional,omitempty"`
	OddsAmerican    *string   `json:"odds_american,omitempty"`
	Status          string    `json:"status"`
	Result          *string   `json:"result,omitempty"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

// BetStatusOpen is the initial state for placed sportsbook bets.
const BetStatusOpen BetStatus = "open"

// SportsBetRecord represents a sports_bets row.
type SportsBetRecord struct {
	ID                 uuid.UUID       `json:"id"`
	PlayerID           uuid.UUID       `json:"player_id"`
	EventID            uuid.UUID       `json:"event_id"`
	MarketID           uuid.UUID       `json:"market_id"`
	SelectionID        uuid.UUID       `json:"selection_id"`
	StakeAmountMinor   int             `json:"stake_amount_minor"`
	Currency           string          `json:"currency"`
	OddsAtPlacement    int             `json:"odds_at_placement"`
	PotentialPayoutMinor int           `json:"potential_payout_minor"`
	Status             BetStatus `json:"status"`
	PayoutAmountMinor  int             `json:"payout_amount_minor"`
	GameRoundID        string          `json:"game_round_id"`
	TransactionID      *uuid.UUID      `json:"transaction_id,omitempty"`
	PlacedAt           time.Time       `json:"placed_at"`
	SettledAt          *time.Time      `json:"settled_at,omitempty"`
}
