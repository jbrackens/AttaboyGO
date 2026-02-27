package infra

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer wraps a kafka-go writer for publishing messages.
type KafkaProducer struct {
	writer  *kafka.Writer
	logger  *slog.Logger
	enabled bool
}

// NewKafkaProducer creates a Kafka producer. If brokers is empty or disabled, writes are no-ops.
func NewKafkaProducer(brokers string, enabled bool, logger *slog.Logger) *KafkaProducer {
	if !enabled || brokers == "" {
		logger.Info("kafka producer disabled")
		return &KafkaProducer{enabled: false, logger: logger}
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	logger.Info("kafka producer initialized", "brokers", brokers)
	return &KafkaProducer{writer: w, logger: logger, enabled: true}
}

// Publish sends a message to the given topic. No-op if disabled.
func (p *KafkaProducer) Publish(ctx context.Context, topic string, key, value []byte) error {
	if !p.enabled {
		return nil
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

// Close shuts down the Kafka writer.
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// KafkaConsumer wraps a kafka-go reader for consuming messages.
type KafkaConsumer struct {
	reader  *kafka.Reader
	logger  *slog.Logger
	enabled bool
}

// NewKafkaConsumer creates a Kafka consumer for the given topic and group.
func NewKafkaConsumer(brokers, topic, groupID string, enabled bool, logger *slog.Logger) *KafkaConsumer {
	if !enabled || brokers == "" {
		return &KafkaConsumer{enabled: false, logger: logger}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(brokers, ","),
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &KafkaConsumer{reader: r, logger: logger, enabled: true}
}

// ReadMessage reads the next message from the consumer. Blocks until a message is available.
func (c *KafkaConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

// Close shuts down the Kafka reader.
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
