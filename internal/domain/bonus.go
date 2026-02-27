package domain

import (
	"time"

	"github.com/google/uuid"
)

// BonusStatus tracks the lifecycle of a player bonus.
type BonusStatus string

const (
	BonusStatusActive    BonusStatus = "active"
	BonusStatusCompleted BonusStatus = "completed"
	BonusStatusExpired   BonusStatus = "expired"
	BonusStatusForfeited BonusStatus = "forfeited"
)

// Bonus represents a bonus definition.
type Bonus struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Code                string    `json:"code"`
	WageringMultiplier  float64   `json:"wagering_multiplier"`
	MinDeposit          int64     `json:"min_deposit"`
	MaxBonus            int64     `json:"max_bonus"`
	DaysUntilExpiry     int       `json:"days_until_expiry"`
	Active              bool      `json:"active"`
}

// PlayerBonus tracks a specific player's bonus instance.
type PlayerBonus struct {
	ID                  uuid.UUID   `json:"id"`
	PlayerID            uuid.UUID   `json:"player_id"`
	BonusID             uuid.UUID   `json:"bonus_id"`
	Status              BonusStatus `json:"status"`
	InitialAmount       int64       `json:"initial_amount"`
	WageringRequirement int64       `json:"wagering_requirement"`
	Wagered             int64       `json:"wagered"`
	ExpiresAt           *time.Time  `json:"expires_at,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
}

// IsWageringComplete checks if the wagering requirement has been met.
func (pb *PlayerBonus) IsWageringComplete() bool {
	return pb.Wagered >= pb.WageringRequirement
}
