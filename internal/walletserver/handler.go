package walletserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter builds the wallet server chi.Router with all provider endpoints.
func NewRouter(
	pool *pgxpool.Pool,
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	bsAdapter *provider.BetSolutionsAdapter,
	ppAdapter *provider.PragmaticAdapter,
	logger *slog.Logger,
) chi.Router {
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
		r.Post("/balance", WalletHandler(bsAdapter, provider.WalletActionBalance, pool, eng, txRepo, logger))
		r.Post("/bet", WalletHandler(bsAdapter, provider.WalletActionBet, pool, eng, txRepo, logger))
		r.Post("/win", WalletHandler(bsAdapter, provider.WalletActionWin, pool, eng, txRepo, logger))
		r.Post("/rollback", WalletHandler(bsAdapter, provider.WalletActionRollback, pool, eng, txRepo, logger))
	})

	// Pragmatic Play endpoints
	r.Route("/pragmatic", func(r chi.Router) {
		r.Post("/", PragmaticHandler(ppAdapter, pool, eng, txRepo, logger))
	})

	return r
}

// WalletHandler creates an HTTP handler for BetSolutions wallet callbacks.
func WalletHandler(
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

		balance, _, err := DispatchWalletAction(r.Context(), pool, eng, txRepo, cb, "betsolutions", logger)
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

// PragmaticHandler creates an HTTP handler for Pragmatic Play wallet callbacks.
func PragmaticHandler(
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

		balance, bonusBalance, err := DispatchWalletAction(r.Context(), pool, eng, txRepo, cb, "pragmatic", logger)
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

// DispatchWalletAction executes the appropriate ledger command for a wallet callback.
func DispatchWalletAction(
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

func handleBalance(ctx context.Context, tx pgx.Tx, eng *ledger.Engine, cb *provider.WalletCallback) (int64, int64, error) {
	player, err := eng.LockPlayerForUpdate(ctx, tx, cb.PlayerID)
	if err != nil {
		return 0, 0, err
	}
	return player.Balance, player.BonusBalance, nil
}

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

func handleRollback(
	ctx context.Context,
	tx pgx.Tx,
	eng *ledger.Engine,
	txRepo repository.TransactionRepository,
	cb *provider.WalletCallback,
	manufacturerID string,
	logger *slog.Logger,
) (int64, int64, error) {
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
