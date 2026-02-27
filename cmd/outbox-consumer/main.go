package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/attaboy/platform/internal/infra"
	"github.com/attaboy/platform/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("outbox consumer failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := infra.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := infra.NewPostgresPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	logger.Info("outbox-consumer connected to postgres")

	pollInterval := 2 * time.Second
	if s := os.Getenv("OUTBOX_POLL_INTERVAL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			pollInterval = d
		}
	}

	batchSize := 100
	if s := os.Getenv("OUTBOX_BATCH_SIZE"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			batchSize = n
		}
	}

	repo := repository.NewOutboxRepository()
	logger.Info("outbox-consumer starting", "poll_interval", pollInterval, "batch_size", batchSize)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("outbox-consumer shutting down")
			return nil
		case <-ticker.C:
			if err := poll(ctx, pool, repo, logger, batchSize); err != nil {
				logger.Error("poll error", "error", err)
			}
		}
	}
}

func poll(ctx context.Context, pool *pgxpool.Pool, repo repository.OutboxRepository, logger *slog.Logger, limit int) error {
	rows, err := repo.FetchUnpublishedRows(ctx, pool, limit)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		logger.Info("outbox event",
			"seq_id", row.SeqID,
			"event_id", row.EventID,
			"aggregate_type", row.AggregateType,
			"event_type", row.EventType,
			"aggregate_id", row.AggregateID,
		)
		ids = append(ids, row.SeqID)
	}

	if err := repo.MarkPublished(ctx, pool, ids); err != nil {
		return fmt.Errorf("mark published: %w", err)
	}

	logger.Info("processed outbox batch", "count", len(ids))
	return nil
}
