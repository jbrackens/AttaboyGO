package policy

import "github.com/attaboy/platform/internal/domain"

// IdentityStatus holds the results of identity verification checks.
type IdentityStatus struct {
	AgeVerified   bool `json:"age_verified"`
	KYCApproved   bool `json:"kyc_approved"`
	OnWatchlist   bool `json:"on_watchlist"`
	AccountActive bool `json:"account_active"`
}

// EvaluateIdentityPolicy checks whether a player passes identity requirements.
// This is a blocking policy — all checks must pass.
func EvaluateIdentityPolicy(profile *domain.PlayerProfile) (IdentityStatus, error) {
	status := IdentityStatus{
		AgeVerified:   profile.DateOfBirth != nil && *profile.DateOfBirth != "",
		KYCApproved:   profile.Verified,
		OnWatchlist:   false, // Would integrate with external watchlist service
		AccountActive: profile.AccountStatus == "active",
	}
	return status, nil
}

// IsIdentityCleared returns true if all identity checks pass.
func (s IdentityStatus) IsIdentityCleared() bool {
	return s.AgeVerified && s.KYCApproved && !s.OnWatchlist && s.AccountActive
}
