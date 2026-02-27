package domain

import (
	"time"

	"github.com/google/uuid"
)

// Balances represents the 3-column balance model (integer cents, numeric(15,0)).
type Balances struct {
	Balance         int64 `json:"balance"`
	BonusBalance    int64 `json:"bonus_balance"`
	ReservedBalance int64 `json:"reserved_balance"`
}

// Player represents a v2_players row.
type Player struct {
	ID        uuid.UUID `json:"id"`
	Balances
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlayerProfile holds the extended player profile from player_profiles.
type PlayerProfile struct {
	PlayerID      uuid.UUID `json:"player_id"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name,omitempty"`
	LastName      string    `json:"last_name,omitempty"`
	DateOfBirth   *string   `json:"date_of_birth,omitempty"`
	Country       string    `json:"country,omitempty"`
	Currency      string    `json:"currency"`
	Language      string    `json:"language"`
	MobilePhone   string    `json:"mobile_phone,omitempty"`
	Address       string    `json:"address,omitempty"`
	PostCode      string    `json:"post_code,omitempty"`
	Verified      bool      `json:"verified"`
	AccountStatus string    `json:"account_status"`
	RiskProfile   string    `json:"risk_profile"`
	CreatedAt     time.Time `json:"created_at"`
}

// AuthUser holds credentials from auth_users.
type AuthUser struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents a player session.
type Session struct {
	ID        uuid.UUID `json:"id"`
	PlayerID  uuid.UUID `json:"player_id"`
	Token     string    `json:"token"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
