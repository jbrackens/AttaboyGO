package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NewTransactionPostedEvent creates the standard wallet event for a ledger entry.
func NewTransactionPostedEvent(tx *Transaction) OutboxDraft {
	payload, _ := json.Marshal(tx)
	return OutboxDraft{
		EventID:       uuid.New(),
		AggregateType: AggregateWallet,
		AggregateID:   tx.PlayerID.String(),
		EventType:     EventTransactionPosted,
		PartitionKey:  tx.PlayerID.String(),
		Headers:       json.RawMessage(`{}`),
		Payload:       payload,
		OccurredAt:    time.Now(),
	}
}

// NewPlayerCreatedEvent creates a player lifecycle event.
func NewPlayerCreatedEvent(playerID uuid.UUID, email, currency string) OutboxDraft {
	payload, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"email":     email,
		"currency":  currency,
	})
	return OutboxDraft{
		EventID:       uuid.New(),
		AggregateType: AggregatePlayer,
		AggregateID:   playerID.String(),
		EventType:     EventPlayerCreated,
		PartitionKey:  playerID.String(),
		Headers:       json.RawMessage(`{}`),
		Payload:       payload,
		OccurredAt:    time.Now(),
	}
}

// NewSelfExclusionEvent creates a responsible gaming self-exclusion event.
func NewSelfExclusionEvent(playerID uuid.UUID, enabled bool, reason string) OutboxDraft {
	evtType := EventSelfExclusionEnabled
	if !enabled {
		evtType = EventSelfExclusionDisabled
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"player_id": playerID.String(),
		"enabled":   enabled,
		"reason":    reason,
	})
	return OutboxDraft{
		EventID:       uuid.New(),
		AggregateType: AggregatePlayer,
		AggregateID:   playerID.String(),
		EventType:     evtType,
		PartitionKey:  playerID.String(),
		Headers:       json.RawMessage(`{}`),
		Payload:       payload,
		OccurredAt:    time.Now(),
	}
}

// NewWalletRouteEvent creates a wallet routing policy event.
func NewWalletRouteEvent(playerID uuid.UUID, accepted bool, source, reason string) OutboxDraft {
	evtType := EventWalletRouteAccepted
	if !accepted {
		evtType = EventWalletRouteRejected
	}
	payload, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"source":    source,
		"reason":    reason,
	})
	return OutboxDraft{
		EventID:       uuid.New(),
		AggregateType: AggregateWallet,
		AggregateID:   playerID.String(),
		EventType:     evtType,
		PartitionKey:  playerID.String(),
		Headers:       json.RawMessage(`{}`),
		Payload:       payload,
		OccurredAt:    time.Now(),
	}
}

// NewLimitBreachedEvent creates a responsible gaming limit breach event.
func NewLimitBreachedEvent(playerID uuid.UUID, limitType string, limitValue, requestedAmount int64) OutboxDraft {
	payload, _ := json.Marshal(map[string]interface{}{
		"player_id":        playerID.String(),
		"limit_type":       limitType,
		"limit_value":      limitValue,
		"requested_amount": requestedAmount,
	})
	return OutboxDraft{
		EventID:       uuid.New(),
		AggregateType: AggregatePlayer,
		AggregateID:   playerID.String(),
		EventType:     EventLimitBreached,
		PartitionKey:  playerID.String(),
		Headers:       json.RawMessage(`{}`),
		Payload:       payload,
		OccurredAt:    time.Now(),
	}
}
