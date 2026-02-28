//go:build integration

package testutil

import (
	"context"
	"time"
)

// CleanAll truncates all tables in dependency-safe order.
func (env *TestEnv) CleanAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Truncate all tables in reverse-dependency order using CASCADE.
	// This is safe for tests and much simpler than ordering manually.
	tables := []string{
		// Affiliate system (many FKs)
		"affiliate_activity_log",
		"affiliate_sub_affiliates",
		"affiliate_media",
		"affiliate_notifications",
		"affiliate_reports_cache",
		"affiliate_payments",
		"affiliate_invoice_lines",
		"affiliate_invoices",
		"affiliate_commissions",
		"affiliate_fees",
		"affiliate_player_refs",
		"affiliate_clicks",
		"affiliate_links",
		"affiliate_deals",
		"affiliate_plan_tiers",
		"affiliate_plans",
		"affiliate_tokens",
		"affiliates",
		"affiliate_brands",

		// Innovation features
		"ai_messages",
		"ai_conversations",
		"prediction_stakes",
		"prediction_markets",
		"social_posts",
		"video_sessions",
		"plugin_dispatches",
		"plugins",

		// Gamification
		"reward_grants",
		"player_quest_progress",
		"quests",
		"player_engagement",

		// Sportsbook
		"odds88_bet_map",
		"odds88_player_map",
		"odds88_feed_state",
		"sports_parlay_legs",
		"sports_parlay_bets",
		"sports_bets",
		"sports_selections",
		"sports_markets",
		"sports_events",
		"sports",

		// Admin
		"player_notes",
		"admin_users",

		// Payments
		"payment_events",
		"payments",
		"payment_methods",

		// Player bonuses
		"player_bonuses",
		"bonuses",

		// Core
		"player_limits",
		"sessions",
		"games",
		"game_manufacturers",
		"event_outbox",
		"v2_transactions",
		"player_profiles",
		"auth_users",
		"v2_players",

		// Dome
		"dome_feed_state",

		// Security
		"login_attempts",
		"password_reset_tokens",
	}

	for _, table := range tables {
		_, _ = env.Pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE")
	}
}
