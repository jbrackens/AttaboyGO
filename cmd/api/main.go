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

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/handler"
	adminhandler "github.com/attaboy/platform/internal/handler/admin"
	"github.com/attaboy/platform/internal/infra"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/provider"
	"github.com/attaboy/platform/internal/repository"
	"github.com/attaboy/platform/internal/service"
	"github.com/go-chi/chi/v5"
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

	// Initialize dependencies
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, playerExpiry, adminExpiry, affiliateExpiry)

	// Repositories
	playerRepo := repository.NewPlayerRepository()
	txRepo := repository.NewTransactionRepository()
	outboxRepo := repository.NewOutboxRepository()
	authUserRepo := repository.NewPgAuthUserRepository()
	profileRepo := repository.NewPgProfileRepository()
	paymentRepo := repository.NewPaymentRepository()

	// Ledger engine
	ledgerEngine := ledger.NewEngine(playerRepo, txRepo, outboxRepo)

	// External providers
	stripeProvider := provider.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret)

	// Services
	authSvc := service.NewAuthService(pool, authUserRepo, playerRepo, profileRepo, jwtMgr)
	paymentSvc := service.NewPaymentService(pool, stripeProvider, paymentRepo, playerRepo, ledgerEngine, logger)
	sportsbookSvc := service.NewSportsbookService(pool, ledgerEngine, logger)
	affiliateSvc := service.NewAffiliateService(pool, jwtMgr, logger)
	pluginSvc := service.NewPluginService(pool, logger)

	// External providers
	rngClient := provider.NewRandomOrgClient(cfg.RandomOrgAPIKey, logger)
	slotopolClient := provider.NewSlotopolClient("http://localhost:4002", logger)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	playerHandler := handler.NewPlayerHandler(playerRepo, profileRepo, pool)
	walletHandler := handler.NewWalletHandler(playerRepo, txRepo, pool)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)
	webhookHandler := handler.NewWebhookHandler(paymentSvc, logger)
	sportsbookHandler := handler.NewSportsbookHandler(sportsbookSvc, pool)
	affiliateHandler := handler.NewAffiliateHandler(affiliateSvc)
	questHandler := handler.NewQuestHandler(pool)
	engagementHandler := handler.NewEngagementHandler(pool)
	pluginHandler := handler.NewPluginHandler(pluginSvc)
	predictionHandler := handler.NewPredictionHandler(pool)
	aiHandler := handler.NewAIHandler(pool)
	videoHandler := handler.NewVideoHandler(pool)
	socialHandler := handler.NewSocialHandler(pool)
	rngHandler := handler.NewRNGHandler(rngClient, slotopolClient)

	// Admin handlers
	playerAdmin := adminhandler.NewPlayerAdminHandler(pool, playerRepo, profileRepo)
	bonusAdmin := adminhandler.NewBonusAdminHandler(pool)
	sbAdmin := adminhandler.NewSportsbookAdminHandler(pool)
	reportsAdmin := adminhandler.NewReportsHandler(pool)
	affiliateAdmin := adminhandler.NewAffiliateAdminHandler(pool)
	questAdmin := adminhandler.NewQuestAdminHandler(pool)
	moderationAdmin := adminhandler.NewModerationHandler(pool)

	// Router
	r := chi.NewRouter()

	// Global middleware (order matters)
	r.Use(handler.Recovery(logger))
	r.Use(handler.RequestID)
	r.Use(handler.RequestLogger(logger))
	r.Use(handler.CORS)
	r.Use(handler.JSONContentType)

	// Health (no auth)
	r.Get("/health", handler.HealthHandler(pool))

	// Webhooks (no auth, no JSON content-type — raw body required for signature verification)
	r.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)

	// Auth routes (no auth)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// Affiliate auth routes (no player auth)
	r.Route("/affiliates", func(r chi.Router) {
		r.Post("/register", affiliateHandler.Register)
		r.Post("/login", affiliateHandler.Login)
	})

	// Public click tracking (no auth)
	r.Get("/track/{btag}", affiliateHandler.TrackClick)

	// Player-authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthenticatePlayer(jwtMgr))

		r.Get("/players/me", playerHandler.GetMe)

		r.Route("/wallet", func(r chi.Router) {
			r.Get("/balance", walletHandler.GetBalance)
			r.Get("/transactions", walletHandler.GetTransactions)
		})

		r.Route("/payments", func(r chi.Router) {
			r.Post("/deposit", paymentHandler.InitiateDeposit)
			r.Post("/withdraw", paymentHandler.RequestWithdrawal)
			r.Get("/history", paymentHandler.GetPaymentHistory)
		})

		r.Route("/sportsbook", func(r chi.Router) {
			r.Get("/sports", sportsbookHandler.ListSports)
			r.Get("/sports/{sportID}/events", sportsbookHandler.ListEvents)
			r.Get("/events/{eventID}/markets", sportsbookHandler.ListMarkets)
			r.Get("/markets/{marketID}/selections", sportsbookHandler.ListSelections)
			r.Post("/bets", sportsbookHandler.PlaceBet)
			r.Get("/bets/me", sportsbookHandler.MyBets)
		})

		r.Route("/quests", func(r chi.Router) {
			r.Get("/", questHandler.ListActive)
			r.Post("/{id}/claim", questHandler.ClaimReward)
		})

		r.Route("/engagement", func(r chi.Router) {
			r.Get("/me", engagementHandler.GetMyEngagement)
			r.Post("/signal", engagementHandler.RecordSignal)
		})

		r.Route("/predictions", func(r chi.Router) {
			r.Get("/markets", predictionHandler.ListMarkets)
			r.Get("/markets/{id}", predictionHandler.GetMarket)
			r.Post("/markets/{id}/stake", predictionHandler.PlaceStake)
			r.Get("/positions", predictionHandler.MyPositions)
		})

		r.Route("/ai", func(r chi.Router) {
			r.Post("/conversations", aiHandler.CreateConversation)
			r.Get("/conversations", aiHandler.ListConversations)
			r.Post("/conversations/{id}/messages", aiHandler.SendMessage)
			r.Get("/conversations/{id}/messages", aiHandler.GetMessages)
		})

		r.Route("/video", func(r chi.Router) {
			r.Post("/sessions", videoHandler.StartSession)
			r.Post("/sessions/{id}/end", videoHandler.EndSession)
			r.Get("/sessions", videoHandler.ListSessions)
		})

		r.Route("/social", func(r chi.Router) {
			r.Post("/posts", socialHandler.CreatePost)
			r.Get("/posts", socialHandler.ListPosts)
			r.Delete("/posts/{id}", socialHandler.DeletePost)
		})

		r.Route("/plugins", func(r chi.Router) {
			r.Get("/", pluginHandler.ListPlugins)
			r.Post("/dispatch", pluginHandler.Dispatch)
			r.Get("/{pluginID}/dispatches", pluginHandler.ListDispatches)
		})

		r.Post("/rng/random", rngHandler.GetRandom)

		r.Route("/slots", func(r chi.Router) {
			r.Get("/games", rngHandler.ListSlotGames)
			r.Post("/spin", rngHandler.Spin)
		})
	})

	// Admin-authenticated routes
	r.Route("/admin", func(r chi.Router) {
		r.Use(auth.AuthenticateAdmin(jwtMgr))

		r.Route("/players", func(r chi.Router) {
			r.Get("/", playerAdmin.SearchPlayers)
			r.Get("/{id}", playerAdmin.GetPlayerDetail)
			r.Patch("/{id}/status", playerAdmin.UpdatePlayerStatus)
		})

		r.Route("/bonuses", func(r chi.Router) {
			r.Get("/", bonusAdmin.ListBonuses)
			r.Post("/", bonusAdmin.CreateBonus)
			r.Patch("/{id}/status", bonusAdmin.UpdateBonusStatus)
		})

		r.Route("/sportsbook", func(r chi.Router) {
			r.Get("/events", sbAdmin.ListEvents)
			r.Post("/events", sbAdmin.CreateEvent)
			r.Patch("/events/{id}/status", sbAdmin.UpdateEventStatus)
		})

		r.Route("/reports", func(r chi.Router) {
			r.Get("/dashboard", reportsAdmin.GetDashboardStats)
			r.Get("/transactions", reportsAdmin.GetTransactionReport)
		})

		r.Route("/affiliates", func(r chi.Router) {
			r.Get("/", affiliateAdmin.ListAffiliates)
			r.Patch("/{id}/status", affiliateAdmin.UpdateAffiliateStatus)
		})

		r.Route("/quests", func(r chi.Router) {
			r.Get("/", questAdmin.ListQuests)
			r.Post("/", questAdmin.CreateQuest)
			r.Patch("/{id}/toggle", questAdmin.ToggleQuest)
		})

		r.Route("/moderation", func(r chi.Router) {
			r.Get("/posts", moderationAdmin.ListFlaggedPosts)
			r.Delete("/posts/{id}", moderationAdmin.DeletePost)
			r.Get("/dispatches", moderationAdmin.ListPluginDispatches)
		})
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
