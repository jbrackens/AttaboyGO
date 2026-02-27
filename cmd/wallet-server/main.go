package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/attaboy/platform/internal/infra"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/attaboy/platform/internal/walletserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("wallet server failed", "error", err)
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
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	pool, err := infra.NewPostgresPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	logger.Info("wallet-server connected to postgres")

	// Repositories & ledger
	playerRepo := repository.NewPlayerRepository()
	txRepo := repository.NewTransactionRepository()
	outboxRepo := repository.NewOutboxRepository()
	ledgerEngine := ledger.NewEngine(playerRepo, txRepo, outboxRepo)

	// Provider adapters
	bsAdapter := provider.NewBetSolutionsAdapter(
		os.Getenv("BETSOLUTIONS_HMAC_SECRET"), logger)
	ppAdapter := provider.NewPragmaticAdapter(
		os.Getenv("PRAGMATIC_SECRET_KEY"), logger)

	// Router
	r := walletserver.NewRouter(pool, ledgerEngine, txRepo, bsAdapter, ppAdapter, logger)

	addr := fmt.Sprintf(":%d", cfg.WalletServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("wallet-server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("wallet-server shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("wallet-server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("wallet-server shutdown failed: %w", err)
	}

	logger.Info("wallet-server stopped gracefully")
	return nil
}
