package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType enumerates all domain event types.
type EventType string

const (
	EventPlayerCreated          EventType = "pam.player.created"
	EventSessionCreated         EventType = "pam.session.created"
	EventSessionRevoked         EventType = "pam.session.revoked"
	EventTransactionPosted      EventType = "pam.wallet.transaction.posted"
	EventWalletRouteAccepted    EventType = "pam.wallet.route.accepted"
	EventWalletRouteRejected    EventType = "pam.wallet.route.rejected"
	EventLimitBreached          EventType = "pam.limit.breached"
	EventSelfExclusionEnabled   EventType = "pam.selfexclusion.enabled"
	EventSelfExclusionDisabled  EventType = "pam.selfexclusion.disabled"
	EventPluginDispatchReq      EventType = "pam.plugin.dispatch.requested"
	EventPluginDispatchDone     EventType = "pam.plugin.dispatch.completed"
	EventPluginDispatchFailed   EventType = "pam.plugin.dispatch.failed"
	EventPluginDispatchFallback EventType = "pam.plugin.dispatch.fallback"
	EventPluginOutputBlocked    EventType = "pam.plugin.output.blocked"
	EventPluginOutputFlagged    EventType = "pam.plugin.output.flagged"
	EventPluginOutputPassed     EventType = "pam.plugin.output.passed"
)

// AggregateType enumerates the aggregate root types for outbox events.
type AggregateType string

const (
	AggregatePlayer  AggregateType = "player"
	AggregateSession AggregateType = "session"
	AggregateWallet  AggregateType = "wallet"
	AggregatePlugin  AggregateType = "plugin"
)

// OutboxDraft is the payload written to the event_outbox table.
// Corresponds to the camelCase-column event_outbox schema (Audit #6).
type OutboxDraft struct {
	EventID       uuid.UUID       `json:"eventId"`
	AggregateType AggregateType   `json:"aggregateType"`
	AggregateID   string          `json:"aggregateId"`
	EventType     EventType       `json:"eventType"`
	PartitionKey  string          `json:"partitionKey"`
	Headers       json.RawMessage `json:"headers"`
	Payload       json.RawMessage `json:"payload"`
	OccurredAt    time.Time       `json:"occurredAt"`
}
