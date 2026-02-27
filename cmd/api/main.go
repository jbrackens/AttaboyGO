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

	"github.com/attaboy/platform/internal/app"
	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/infra"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load config
	cfg, err := infra.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Connect to Postgres
	pool, err := infra.NewPostgresPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	// Parse JWT expiry durations
	playerExpiry, err := time.ParseDuration(cfg.JWTPlayerExpiry)
	if err != nil {
		return fmt.Errorf("parse player JWT expiry: %w", err)
	}
	adminExpiry, err := time.ParseDuration(cfg.JWTAdminExpiry)
	if err != nil {
		return fmt.Errorf("parse admin JWT expiry: %w", err)
	}
	affiliateExpiry, err := time.ParseDuration(cfg.JWTAffiliateExpiry)
	if err != nil {
		return fmt.Errorf("parse affiliate JWT expiry: %w", err)
	}

	// Initialize JWT manager
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, playerExpiry, adminExpiry, affiliateExpiry)

	// Build router via wire
	r := app.NewRouter(app.RouterDeps{
		Pool:                pool,
		JWTMgr:              jwtMgr,
		Logger:              logger,
		StripeSecretKey:     cfg.StripeSecretKey,
		StripeWebhookSecret: cfg.StripeWebhookSecret,
		RandomOrgAPIKey:     cfg.RandomOrgAPIKey,
		SlotopolBaseURL:     "http://localhost:4002",
		CORSAllowedOrigins:  cfg.CORSAllowedOrigins,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.APIPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped gracefully")
	return nil
}
