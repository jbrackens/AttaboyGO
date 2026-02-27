package repository

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

type outboxRepo struct{}

// NewOutboxRepository returns a pgx-backed OutboxRepository.
func NewOutboxRepository() OutboxRepository {
	return &outboxRepo{}
}

// Insert writes an outbox event using the camelCase column names (Audit #6).
func (r *outboxRepo) Insert(ctx context.Context, db DBTX, draft domain.OutboxDraft) error {
	_, err := db.Exec(ctx, `
		INSERT INTO event_outbox
		  ("eventId", "aggregateType", "aggregateId", "eventType", "partitionKey", "headers", "payload", "occurredAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		draft.EventID,
		string(draft.AggregateType),
		draft.AggregateID,
		string(draft.EventType),
		draft.PartitionKey,
		draft.Headers,
		draft.Payload,
		draft.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func (r *outboxRepo) FetchUnpublished(ctx context.Context, db DBTX, limit int) ([]domain.OutboxDraft, error) {
	rows, err := db.Query(ctx, `
		SELECT "id", "eventId", "aggregateType", "aggregateId", "eventType",
		       "partitionKey", "headers", "payload", "occurredAt"
		FROM event_outbox
		ORDER BY "id" ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished events: %w", err)
	}
	defer rows.Close()

	var events []domain.OutboxDraft
	for rows.Next() {
		var d domain.OutboxDraft
		var seqID int64
		err := rows.Scan(&seqID, &d.EventID, &d.AggregateType, &d.AggregateID,
			&d.EventType, &d.PartitionKey, &d.Headers, &d.Payload, &d.OccurredAt)
		if err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		events = append(events, d)
	}
	return events, rows.Err()
}

func (r *outboxRepo) MarkPublished(ctx context.Context, db DBTX, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := db.Exec(ctx, `DELETE FROM event_outbox WHERE "id" = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

// scanOutboxRow is unused currently but reserved for single-row scans.
var _ pgx.Row // keep import
