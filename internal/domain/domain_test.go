package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Validator Tests ---

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
		errMsg  string
	}{
		{"valid email", "user@example.com", false, ""},
		{"valid email with dots", "first.last@example.co.uk", false, ""},
		{"valid email with plus", "user+tag@example.com", false, ""},
		{"valid email with dash", "user-name@exam-ple.com", false, ""},
		{"empty string", "", true, "email is required"},
		{"no at sign", "userexample.com", true, "invalid email format"},
		{"no domain", "user@", true, "invalid email format"},
		{"no user", "@example.com", true, "invalid email format"},
		{"double at", "user@@example.com", true, "invalid email format"},
		{"no tld", "user@example", true, "invalid email format"},
		{"single char tld", "user@example.c", true, "invalid email format"},
		{"spaces", "user @example.com", true, "invalid email format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{"valid EUR", "EUR", false},
		{"valid USD", "USD", false},
		{"valid GBP", "GBP", false},
		{"lowercase", "eur", true},
		{"mixed case", "Eur", true},
		{"too short", "EU", true},
		{"too long", "EURO", true},
		{"empty", "", true},
		{"numbers", "123", true},
		{"with space", "EU ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCurrency(tt.currency)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid currency code")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePositiveAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		wantErr bool
	}{
		{"positive", 100, false},
		{"one cent", 1, false},
		{"large amount", 999_999_999, false},
		{"zero", 0, true},
		{"negative", -100, true},
		{"min int64", -9223372036854775808, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveAmount(tt.amount)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "amount must be positive")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAttestation(t *testing.T) {
	validAttestation := Attestation{
		Provider:      "dome",
		AttestationID: "att-123",
		Digest:        "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", // 32 hex chars
		IssuedAt:      time.Now(),
	}

	t.Run("valid attestation", func(t *testing.T) {
		require.NoError(t, ValidateAttestation(validAttestation))
	})

	t.Run("missing provider", func(t *testing.T) {
		a := validAttestation
		a.Provider = ""
		err := ValidateAttestation(a)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
	})

	t.Run("missing attestation ID", func(t *testing.T) {
		a := validAttestation
		a.AttestationID = ""
		err := ValidateAttestation(a)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "attestation ID is required")
	})

	t.Run("digest too short", func(t *testing.T) {
		a := validAttestation
		a.Digest = "abc123" // < 32 chars
		err := ValidateAttestation(a)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32-128 hex characters")
	})

	t.Run("digest non-hex", func(t *testing.T) {
		a := validAttestation
		a.Digest = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz" // 32 non-hex chars
		err := ValidateAttestation(a)
		require.Error(t, err)
	})

	t.Run("digest 128 chars valid", func(t *testing.T) {
		a := validAttestation
		a.Digest = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
		require.NoError(t, ValidateAttestation(a))
	})

	t.Run("zero issued_at", func(t *testing.T) {
		a := validAttestation
		a.IssuedAt = time.Time{}
		err := ValidateAttestation(a)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuedAt is required")
	})
}

// --- AppError Tests ---

func TestAppError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := ErrNotFound("player", "abc-123")
		assert.Equal(t, "NOT_FOUND: player abc-123 not found", err.Error())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := ErrInternal("database error", cause)
		assert.Contains(t, err.Error(), "INTERNAL_ERROR")
		assert.Contains(t, err.Error(), "connection refused")
	})
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := ErrInternal("wrapped", cause)
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestErrorFactories(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantCode   string
		wantStatus int
	}{
		{"ErrNotFound", ErrNotFound("player", "123"), "NOT_FOUND", 404},
		{"ErrConflict", ErrConflict("already exists"), "CONFLICT", 409},
		{"ErrValidation", ErrValidation("bad input"), "VALIDATION_ERROR", 400},
		{"ErrUnauthorized", ErrUnauthorized("no token"), "UNAUTHORIZED", 401},
		{"ErrForbidden", ErrForbidden("not allowed"), "FORBIDDEN", 403},
		{"ErrInsufficientBalance", ErrInsufficientBalance(), "INSUFFICIENT_BALANCE", 400},
		{"ErrIdempotent", ErrIdempotent("tx-abc"), "IDEMPOTENT", 200},
		{"ErrAccountLocked", ErrAccountLocked("too many attempts"), "ACCOUNT_LOCKED", 429},
		{"ErrInternal", ErrInternal("oops", nil), "INTERNAL_ERROR", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantCode, tt.err.Code)
			assert.Equal(t, tt.wantStatus, tt.err.Status)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

// --- BalanceUpdate Tests ---

func TestBalanceUpdate_HasDelta(t *testing.T) {
	t.Run("all zero", func(t *testing.T) {
		u := BalanceUpdate{}
		assert.False(t, u.HasBalanceDelta())
		assert.False(t, u.HasBonusDelta())
		assert.False(t, u.HasReservedDelta())
	})

	t.Run("balance only", func(t *testing.T) {
		u := BalanceUpdate{Balance: 100}
		assert.True(t, u.HasBalanceDelta())
		assert.False(t, u.HasBonusDelta())
		assert.False(t, u.HasReservedDelta())
	})

	t.Run("bonus only", func(t *testing.T) {
		u := BalanceUpdate{BonusBalance: -50}
		assert.False(t, u.HasBalanceDelta())
		assert.True(t, u.HasBonusDelta())
		assert.False(t, u.HasReservedDelta())
	})

	t.Run("reserved only", func(t *testing.T) {
		u := BalanceUpdate{ReservedBalance: 200}
		assert.False(t, u.HasBalanceDelta())
		assert.False(t, u.HasBonusDelta())
		assert.True(t, u.HasReservedDelta())
	})

	t.Run("all non-zero", func(t *testing.T) {
		u := BalanceUpdate{Balance: 100, BonusBalance: -50, ReservedBalance: 200}
		assert.True(t, u.HasBalanceDelta())
		assert.True(t, u.HasBonusDelta())
		assert.True(t, u.HasReservedDelta())
	})
}

// --- Engagement Score Tests ---

func TestEngagementSignals_ComputeScore(t *testing.T) {
	tests := []struct {
		name    string
		signals EngagementSignals
		want    int
	}{
		{"all zero", EngagementSignals{}, 0},
		{"video only", EngagementSignals{VideoMinutes: 10}, 20},
		{"social only", EngagementSignals{SocialInteractions: 5}, 15},
		{"predictions only", EngagementSignals{PredictionActions: 3}, 15},
		{"mixed signals", EngagementSignals{VideoMinutes: 10, SocialInteractions: 5, PredictionActions: 3}, 50},
		{"large values", EngagementSignals{VideoMinutes: 100, SocialInteractions: 50, PredictionActions: 30}, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.signals.ComputeScore())
		})
	}
}

func TestDefaultGamificationConfig(t *testing.T) {
	cfg := DefaultGamificationConfig()
	assert.Equal(t, 50, cfg.MinEngagementScore)
	assert.Equal(t, 60, cfg.CooldownMinutes)
	assert.Equal(t, int64(250_000), cfg.DailyBudgetCents)
}

// --- RiskTier TTL Tests ---

func TestRiskTierTTL(t *testing.T) {
	tests := []struct {
		tier RiskTier
		want time.Duration
	}{
		{RiskTierHigh, 5 * time.Minute},
		{RiskTierMedium, 30 * time.Minute},
		{RiskTierLow, 1 * time.Hour},
		{"unknown", 1 * time.Hour}, // default case
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			assert.Equal(t, tt.want, RiskTierTTL(tt.tier))
		})
	}
}

// --- PlayerBonus Wagering Tests ---

func TestPlayerBonus_IsWageringComplete(t *testing.T) {
	tests := []struct {
		name     string
		wagered  int64
		required int64
		want     bool
	}{
		{"not complete", 500, 1000, false},
		{"exactly met", 1000, 1000, true},
		{"exceeded", 1500, 1000, true},
		{"zero requirement", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &PlayerBonus{Wagered: tt.wagered, WageringRequirement: tt.required}
			assert.Equal(t, tt.want, pb.IsWageringComplete())
		})
	}
}

// --- CancellationTypeMap Tests ---

func TestCancellationTypeMap(t *testing.T) {
	tests := []struct {
		original TransactionType
		expected TransactionType
		exists   bool
	}{
		{TxDeposit, TxCancelDeposit, true},
		{TxBet, TxCancelBet, true},
		{TxWin, TxCancelWin, true},
		{TxWithdrawal, TxCancelWithdrawal, true},
		{TxBonusCredit, "", false},
		{TxSettlementLoss, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.original), func(t *testing.T) {
			result, ok := CancellationTypeMap[tt.original]
			assert.Equal(t, tt.exists, ok)
			if tt.exists {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// --- Event Factory Tests ---

func TestNewTransactionPostedEvent(t *testing.T) {
	playerID := uuid.New()
	tx := &Transaction{
		ID:       uuid.New(),
		PlayerID: playerID,
		Type:     TxDeposit,
		Amount:   10000,
	}

	event := NewTransactionPostedEvent(tx)

	assert.NotEqual(t, uuid.Nil, event.EventID)
	assert.Equal(t, AggregateWallet, event.AggregateType)
	assert.Equal(t, playerID.String(), event.AggregateID)
	assert.Equal(t, EventTransactionPosted, event.EventType)
	assert.Equal(t, playerID.String(), event.PartitionKey)
	assert.NotEmpty(t, event.Payload)
	assert.False(t, event.OccurredAt.IsZero())

	// Verify payload contains transaction data
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(event.Payload, &payload))
	assert.Equal(t, float64(10000), payload["amount"])
}

func TestNewPlayerCreatedEvent(t *testing.T) {
	playerID := uuid.New()
	event := NewPlayerCreatedEvent(playerID, "test@example.com", "EUR")

	assert.Equal(t, AggregatePlayer, event.AggregateType)
	assert.Equal(t, playerID.String(), event.AggregateID)
	assert.Equal(t, EventPlayerCreated, event.EventType)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(event.Payload, &payload))
	assert.Equal(t, "test@example.com", payload["email"])
	assert.Equal(t, "EUR", payload["currency"])
}

func TestNewSelfExclusionEvent(t *testing.T) {
	playerID := uuid.New()

	t.Run("enabled", func(t *testing.T) {
		event := NewSelfExclusionEvent(playerID, true, "personal choice")
		assert.Equal(t, EventSelfExclusionEnabled, event.EventType)
	})

	t.Run("disabled", func(t *testing.T) {
		event := NewSelfExclusionEvent(playerID, false, "cooldown expired")
		assert.Equal(t, EventSelfExclusionDisabled, event.EventType)
	})
}

func TestNewWalletRouteEvent(t *testing.T) {
	playerID := uuid.New()

	t.Run("accepted", func(t *testing.T) {
		event := NewWalletRouteEvent(playerID, true, "deposit", "within limits")
		assert.Equal(t, EventWalletRouteAccepted, event.EventType)
	})

	t.Run("rejected", func(t *testing.T) {
		event := NewWalletRouteEvent(playerID, false, "deposit", "daily limit exceeded")
		assert.Equal(t, EventWalletRouteRejected, event.EventType)
	})
}

func TestNewLimitBreachedEvent(t *testing.T) {
	playerID := uuid.New()
	event := NewLimitBreachedEvent(playerID, "daily_deposit", 100000, 150000)

	assert.Equal(t, EventLimitBreached, event.EventType)
	assert.Equal(t, AggregatePlayer, event.AggregateType)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(event.Payload, &payload))
	assert.Equal(t, "daily_deposit", payload["limit_type"])
	assert.Equal(t, float64(100000), payload["limit_value"])
	assert.Equal(t, float64(150000), payload["requested_amount"])
}
