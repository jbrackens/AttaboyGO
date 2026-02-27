package domain

import (
	"time"

	"github.com/google/uuid"
)

// CommissionModel enumerates affiliate commission calculation models.
type CommissionModel string

const (
	CommissionCPA      CommissionModel = "cpa"
	CommissionRevShare CommissionModel = "rev_share"
	CommissionHybrid   CommissionModel = "hybrid"
	CommissionTiered   CommissionModel = "tiered"
)

// Affiliate represents an affiliate partner.
type Affiliate struct {
	ID              uuid.UUID       `json:"id"`
	Email           string          `json:"email"`
	PasswordHash    string          `json:"-"`
	CompanyName     string          `json:"company_name,omitempty"`
	ContactName     string          `json:"contact_name"`
	Status          string          `json:"status"` // active, suspended, pending
	CommissionModel CommissionModel `json:"commission_model"`
	CommissionRate  float64         `json:"commission_rate"`
	Btag            string          `json:"btag"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// AffiliateClick tracks a referral click.
type AffiliateClick struct {
	ID          uuid.UUID `json:"id"`
	AffiliateID uuid.UUID `json:"affiliate_id"`
	Btag        string    `json:"btag"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	LandingURL  string    `json:"landing_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Commission represents a calculated affiliate commission.
type Commission struct {
	ID          uuid.UUID `json:"id"`
	AffiliateID uuid.UUID `json:"affiliate_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	Amount      int64     `json:"amount"`
	Model       CommissionModel `json:"model"`
	Period      string    `json:"period"` // e.g. "2026-02"
	Status      string    `json:"status"` // pending, approved, paid
	CreatedAt   time.Time `json:"created_at"`
}

// Invoice represents an affiliate payment invoice.
type Invoice struct {
	ID          uuid.UUID `json:"id"`
	AffiliateID uuid.UUID `json:"affiliate_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"` // draft, sent, paid
	Period      string    `json:"period"`
	CreatedAt   time.Time `json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}
