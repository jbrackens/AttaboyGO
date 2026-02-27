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
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("wallet-server request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr)
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// BetSolutions endpoints
	r.Route("/betsolutions", func(r chi.Router) {
		r.Post("/balance", walletHandler(bsAdapter, provider.WalletActionBalance, pool, ledgerEngine, logger))
		r.Post("/bet", walletHandler(bsAdapter, provider.WalletActionBet, pool, ledgerEngine, logger))
		r.Post("/win", walletHandler(bsAdapter, provider.WalletActionWin, pool, ledgerEngine, logger))
		r.Post("/rollback", walletHandler(bsAdapter, provider.WalletActionRollback, pool, ledgerEngine, logger))
	})

	// Pragmatic Play endpoints
	r.Route("/pragmatic", func(r chi.Router) {
		r.Post("/", pragmaticHandler(ppAdapter, pool, ledgerEngine, logger))
	})

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

// walletHandler creates an HTTP handler for BetSolutions wallet callbacks.
func walletHandler(
	adapter *provider.BetSolutionsAdapter,
	action provider.WalletAction,
	pool *pgxpool.Pool,
	_ *ledger.Engine,
	logger *slog.Logger,
) http.HandlerFunc {
	_ = pool
	return func(w http.ResponseWriter, r *http.Request) {
		req, body, err := adapter.ParseRequest(r)
		if err != nil {
			adapter.RespondJSON(w, provider.BetSolutionsResponse{
				StatusCode: 400,
				Error:      "invalid request",
			})
			return
		}

		if !adapter.VerifySignature(body, req.Hash) {
			logger.Warn("betsolutions signature mismatch")
			adapter.RespondJSON(w, provider.BetSolutionsResponse{
				StatusCode: 401,
				Error:      "invalid signature",
			})
			return
		}

		cb, err := adapter.ToWalletCallback(req, action)
		if err != nil {
			adapter.RespondJSON(w, provider.BetSolutionsResponse{
				StatusCode: 400,
				Error:      err.Error(),
			})
			return
		}

		// Wallet operations would use ledgerEngine here
		// For now, return balance placeholder
		logger.Info("betsolutions callback",
			"action", cb.Action,
			"player_id", cb.PlayerID,
			"amount", cb.Amount,
			"tx_id", cb.TransactionID)

		adapter.RespondJSON(w, provider.BetSolutionsResponse{
			StatusCode: 200,
			Balance:    0, // actual balance from ledger
		})
	}
}

// pragmaticHandler creates an HTTP handler for Pragmatic Play wallet callbacks.
func pragmaticHandler(
	adapter *provider.PragmaticAdapter,
	pool *pgxpool.Pool,
	_ *ledger.Engine,
	logger *slog.Logger,
) http.HandlerFunc {
	_ = pool
	return func(w http.ResponseWriter, r *http.Request) {
		req, body, err := adapter.ParseRequest(r)
		if err != nil {
			adapter.RespondJSON(w, provider.PragmaticResponse{
				Error:   1,
				Message: "invalid request",
			})
			return
		}

		if !adapter.VerifySignature(body, req.ProvidedHash) {
			logger.Warn("pragmatic signature mismatch")
			adapter.RespondJSON(w, provider.PragmaticResponse{
				Error:   1,
				Message: "invalid signature",
			})
			return
		}

		cb, err := adapter.ToWalletCallback(req)
		if err != nil {
			adapter.RespondJSON(w, provider.PragmaticResponse{
				Error:   1,
				Message: err.Error(),
			})
			return
		}

		logger.Info("pragmatic callback",
			"action", cb.Action,
			"player_id", cb.PlayerID,
			"amount", cb.Amount,
			"tx_id", cb.TransactionID)

		adapter.RespondJSON(w, provider.PragmaticResponse{
			Currency: cb.Currency,
			Cash:     "0.00", // actual balance from ledger
			Bonus:    "0.00",
			Error:    0,
		})
	}
}
