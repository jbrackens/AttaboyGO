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

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/infra"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
		r.Post("/balance", walletHandler(bsAdapter, provider.WalletActionBalance, pool, ledgerEngine, txRepo, logger))
		r.Post("/bet", walletHandler(bsAdapter, provider.WalletActionBet, pool, ledgerEngine, txRepo, logger))
		r.Post("/win", walletHandler(bsAdapter, provider.WalletActionWin, pool, ledgerEngine, txRepo, logger))
		r.Post("/rollback", walletHandler(bsAdapter, provider.WalletActionRollback, pool, ledgerEngine, txRepo, logger))
	})

	// Pragmatic Play endpoints
	r.Route("/pragmatic", func(r chi.Router) {
		r.Post("/", pragmaticHandler(ppAdapter, pool, ledgerEngine, txRepo, logger))
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
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	logger *slog.Logger,
) http.HandlerFunc {
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

		logger.Info("betsolutions callback",
			"action", cb.Action,
			"player_id", cb.PlayerID,
			"amount", cb.Amount,
			"tx_id", cb.TransactionID)

		balance, _, err := dispatchWalletAction(r.Context(), pool, eng, txRepo, cb, "betsolutions", logger)
		if err != nil {
			logger.Error("betsolutions wallet action failed", "error", err, "action", cb.Action, "player_id", cb.PlayerID)
			adapter.RespondJSON(w, provider.BetSolutionsResponse{
				StatusCode: 500,
				Error:      "internal error",
			})
			return
		}

		adapter.RespondJSON(w, provider.BetSolutionsResponse{
			StatusCode: 200,
			Balance:    balance,
		})
	}
}

// pragmaticHandler creates an HTTP handler for Pragmatic Play wallet callbacks.
func pragmaticHandler(
	adapter *provider.PragmaticAdapter,
	pool *pgxpool.Pool,
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	logger *slog.Logger,
) http.HandlerFunc {
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

		balance, bonusBalance, err := dispatchWalletAction(r.Context(), pool, eng, txRepo, cb, "pragmatic", logger)
		if err != nil {
			logger.Error("pragmatic wallet action failed", "error", err, "action", cb.Action, "player_id", cb.PlayerID)
			adapter.RespondJSON(w, provider.PragmaticResponse{
				Error:   1,
				Message: "internal error",
			})
			return
		}

		adapter.RespondJSON(w, provider.PragmaticResponse{
			Currency: cb.Currency,
			Cash:     provider.FormatCents(balance),
			Bonus:    provider.FormatCents(bonusBalance),
			Error:    0,
		})
	}
}

// dispatchWalletAction executes the appropriate ledger command for a wallet callback.
func dispatchWalletAction(
	ctx context.Context,
	pool *pgxpool.Pool,
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	cb *provider.WalletCallback,
	manufacturerID string,
	logger *slog.Logger,
) (balance, bonusBalance int64, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	switch cb.Action {
	case provider.WalletActionBalance:
		balance, bonusBalance, err = handleBalance(ctx, tx, eng, cb)
	case provider.WalletActionBet:
		balance, bonusBalance, err = handleBet(ctx, tx, eng, cb, manufacturerID)
	case provider.WalletActionWin:
		balance, bonusBalance, err = handleWin(ctx, tx, eng, cb, manufacturerID)
	case provider.WalletActionRollback:
		balance, bonusBalance, err = handleRollback(ctx, tx, eng, txRepo, cb, manufacturerID, logger)
	default:
		return 0, 0, fmt.Errorf("unknown wallet action: %s", cb.Action)
	}

	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit transaction: %w", err)
	}

	return balance, bonusBalance, nil
}

// handleBalance locks the player and returns current balances (no mutation).
func handleBalance(ctx context.Context, tx pgx.Tx, eng *ledger.Engine, cb *provider.WalletCallback) (int64, int64, error) {
	player, err := eng.LockPlayerForUpdate(ctx, tx, cb.PlayerID)
	if err != nil {
		return 0, 0, err
	}
	return player.Balance, player.BonusBalance, nil
}

// handleBet deducts the bet amount from the player's wallet.
func handleBet(ctx context.Context, tx pgx.Tx, eng *ledger.Engine, cb *provider.WalletCallback, manufacturerID string) (int64, int64, error) {
	result, err := eng.ExecutePlaceBet(ctx, tx, domain.PlaceBetParams{
		PlayerID:              cb.PlayerID,
		Amount:                cb.Amount,
		ExternalTransactionID: cb.TransactionID,
		ManufacturerID:        manufacturerID,
		SubTransactionID:      "1",
		GameRoundID:           cb.RoundID,
	})
	if err != nil {
		return 0, 0, err
	}
	return result.Player.Balance, result.Player.BonusBalance, nil
}

// handleWin credits the win amount to the player's wallet.
func handleWin(ctx context.Context, tx pgx.Tx, eng *ledger.Engine, cb *provider.WalletCallback, manufacturerID string) (int64, int64, error) {
	result, err := eng.ExecuteCreditWin(ctx, tx, domain.CreditWinParams{
		PlayerID:              cb.PlayerID,
		Amount:                cb.Amount,
		ExternalTransactionID: cb.TransactionID,
		ManufacturerID:        manufacturerID,
		SubTransactionID:      "1",
		GameRoundID:           cb.RoundID,
		WinType:               domain.CasinoWinNormal,
	})
	if err != nil {
		return 0, 0, err
	}
	return result.Player.Balance, result.Player.BonusBalance, nil
}

// handleRollback cancels the original transaction identified by the provider's external ID.
func handleRollback(
	ctx context.Context,
	tx pgx.Tx,
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	cb *provider.WalletCallback,
	manufacturerID string,
	logger *slog.Logger,
) (int64, int64, error) {
	// Find the original transaction by the provider's external transaction ID.
	original, err := txRepo.FindExisting(ctx, tx, domain.IdempotencyKey{
		PlayerID:              cb.PlayerID,
		ManufacturerID:        manufacturerID,
		ExternalTransactionID: cb.TransactionID,
		SubTransactionID:      "1",
	})
	if err != nil {
		return 0, 0, fmt.Errorf("find original transaction: %w", err)
	}

	if original == nil {
		// Provider is rolling back a transaction we never recorded.
		// Return current balance as an idempotent success.
		logger.Warn("rollback: original transaction not found, returning current balance",
			"player_id", cb.PlayerID,
			"external_tx_id", cb.TransactionID,
			"manufacturer", manufacturerID)
		player, lockErr := eng.LockPlayerForUpdate(ctx, tx, cb.PlayerID)
		if lockErr != nil {
			return 0, 0, lockErr
		}
		return player.Balance, player.BonusBalance, nil
	}

	rollbackExtID := fmt.Sprintf("rollback_%s", cb.TransactionID)
	result, err := eng.ExecuteCancelTransaction(ctx, tx, domain.CancelTransactionParams{
		PlayerID:              cb.PlayerID,
		Amount:                original.Amount,
		ExternalTransactionID: rollbackExtID,
		ManufacturerID:        manufacturerID,
		SubTransactionID:      "1",
		TargetTransactionID:   original.ID,
	})
	if err != nil {
		return 0, 0, err
	}
	return result.Player.Balance, result.Player.BonusBalance, nil
}
