package infra

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxPoller polls the event_outbox table and publishes events to Kafka.
type OutboxPoller struct {
	pool      *pgxpool.Pool
	producer  *KafkaProducer
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
}

// NewOutboxPoller creates a new outbox poller.
func NewOutboxPoller(pool *pgxpool.Pool, producer *KafkaProducer, logger *slog.Logger) *OutboxPoller {
	return &OutboxPoller{
		pool:      pool,
		producer:  producer,
		logger:    logger,
		interval:  500 * time.Millisecond,
		batchSize: 100,
	}
}

// Start begins polling in a goroutine. Stops when ctx is cancelled.
func (p *OutboxPoller) Start(ctx context.Context) {
	p.logger.Info("outbox poller started", "interval", p.interval, "batch_size", p.batchSize)

	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				p.logger.Info("outbox poller stopped")
				return
			case <-ticker.C:
				if err := p.poll(ctx); err != nil {
					p.logger.Error("outbox poll error", "error", err)
				}
			}
		}
	}()
}

func (p *OutboxPoller) poll(ctx context.Context) error {
	// Fetch unpublished events
	rows, err := p.pool.Query(ctx, `
		SELECT "eventId", "aggregateType", "aggregateId", "eventType", "payload", "occurredAt"
		FROM event_outbox
		WHERE "publishedAt" IS NULL
		ORDER BY "occurredAt" ASC
		LIMIT $1`, p.batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	type outboxEvent struct {
		EventID       uuid.UUID
		AggregateType string
		AggregateID   string
		EventType     string
		Payload       json.RawMessage
		OccurredAt    time.Time
	}

	var events []outboxEvent
	for rows.Next() {
		var e outboxEvent
		if err := rows.Scan(&e.EventID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.OccurredAt); err != nil {
			return err
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		return nil
	}

	// Publish each event to Kafka
	for _, e := range events {
		topic := "attaboy." + e.AggregateType + "." + e.EventType
		key := []byte(e.AggregateID)

		msg, _ := json.Marshal(map[string]interface{}{
			"event_id":       e.EventID,
			"aggregate_type": e.AggregateType,
			"aggregate_id":   e.AggregateID,
			"event_type":     e.EventType,
			"payload":        e.Payload,
			"occurred_at":    e.OccurredAt,
		})

		if err := p.producer.Publish(ctx, topic, key, msg); err != nil {
			p.logger.Error("kafka publish failed", "event_id", e.EventID, "error", err)
			continue
		}

		// Mark as published
		_, err := p.pool.Exec(ctx,
			`UPDATE event_outbox SET "publishedAt" = now() WHERE "eventId" = $1`, e.EventID)
		if err != nil {
			p.logger.Error("mark published failed", "event_id", e.EventID, "error", err)
		}
	}

	p.logger.Debug("outbox poll complete", "published", len(events))
	return nil
}
