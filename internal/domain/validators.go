package domain

import (
	"fmt"
	"regexp"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)
	digestRegex   = regexp.MustCompile(`^[0-9a-fA-F]{32,128}$`)
)

// ValidateEmail checks if an email address is valid.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateCurrency checks if a currency code is ISO 4217.
func ValidateCurrency(currency string) error {
	if !currencyRegex.MatchString(currency) {
		return fmt.Errorf("invalid currency code: %s", currency)
	}
	return nil
}

// ValidatePositiveAmount checks that an amount is positive (in cents).
func ValidatePositiveAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive, got %d", amount)
	}
	return nil
}

// ValidateAttestation checks the oracle attestation fields.
func ValidateAttestation(a Attestation) error {
	if a.Provider == "" {
		return fmt.Errorf("attestation provider is required")
	}
	if a.AttestationID == "" {
		return fmt.Errorf("attestation ID is required")
	}
	if !digestRegex.MatchString(a.Digest) {
		return fmt.Errorf("attestation digest must be 32-128 hex characters")
	}
	if a.IssuedAt.IsZero() {
		return fmt.Errorf("attestation issuedAt is required")
	}
	return nil
}
