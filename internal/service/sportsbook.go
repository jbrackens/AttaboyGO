package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SportsbookService handles sportsbook bet operations.
type SportsbookService struct {
	pool   *pgxpool.Pool
	engine *ledger.Engine
	logger *slog.Logger
}

// NewSportsbookService creates a SportsbookService.
func NewSportsbookService(pool *pgxpool.Pool, engine *ledger.Engine, logger *slog.Logger) *SportsbookService {
	return &SportsbookService{pool: pool, engine: engine, logger: logger}
}

// PlaceBetInput holds the bet placement request.
type PlaceBetInput struct {
	EventID     uuid.UUID `json:"event_id"`
	MarketID    uuid.UUID `json:"market_id"`
	SelectionID uuid.UUID `json:"selection_id"`
	Stake       int64     `json:"stake"`
}

// PlaceBetResult holds the result of a bet placement.
type PlaceBetResult struct {
	BetID       uuid.UUID `json:"bet_id"`
	GameRoundID string    `json:"game_round_id"`
	Stake       int64     `json:"stake"`
	Odds        int       `json:"odds"`
	PotentialPayout int64 `json:"potential_payout"`
}

// PlaceBet places a single bet, deducting from the player's wallet.
func (s *SportsbookService) PlaceBet(ctx context.Context, playerID uuid.UUID, input PlaceBetInput) (*PlaceBetResult, error) {
	if input.Stake <= 0 {
		return nil, domain.ErrValidation("stake must be positive")
	}

	// Fetch selection for odds
	var odds int
	err := s.pool.QueryRow(ctx,
		`SELECT odds_decimal FROM sports_selections WHERE id = $1 AND status = 'active'`,
		input.SelectionID).Scan(&odds)
	if err != nil {
		return nil, domain.ErrNotFound("selection", input.SelectionID.String())
	}

	// Calculate potential payout: stake * (odds / 100)
	potentialPayout := input.Stake * int64(odds) / 100

	betID := uuid.New()
	gameRoundID := fmt.Sprintf("sb_%s", betID.String()[:8])

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, domain.ErrInternal("begin tx", err)
	}
	defer tx.Rollback(ctx)

	// Deduct from wallet via ledger
	extTxID := fmt.Sprintf("bet_%s", betID.String()[:8])
	result, err := s.engine.ExecutePlaceBet(ctx, tx, domain.PlaceBetParams{
		PlayerID:              playerID,
		Amount:                input.Stake,
		ExternalTransactionID: extTxID,
		ManufacturerID:        "sportsbook",
		SubTransactionID:      "1",
		GameRoundID:           gameRoundID,
		Metadata:              json.RawMessage(fmt.Sprintf(`{"event_id":"%s","market_id":"%s","selection_id":"%s"}`, input.EventID, input.MarketID, input.SelectionID)),
	})
	if err != nil {
		return nil, err
	}

	// Insert bet record
	_, err = tx.Exec(ctx, `
		INSERT INTO sports_bets (id, player_id, event_id, market_id, selection_id,
			stake_amount_minor, currency, odds_at_placement, potential_payout_minor,
			status, game_round_id, transaction_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		betID, playerID, input.EventID, input.MarketID, input.SelectionID,
		input.Stake, "EUR", odds, potentialPayout,
		"open", gameRoundID, result.Transaction.ID,
	)
	if err != nil {
		return nil, domain.ErrInternal("insert bet", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal("commit tx", err)
	}

	return &PlaceBetResult{
		BetID:           betID,
		GameRoundID:     gameRoundID,
		Stake:           input.Stake,
		Odds:            odds,
		PotentialPayout: potentialPayout,
	}, nil
}

// ListPlayerBets returns a player's bet history.
func (s *SportsbookService) ListPlayerBets(ctx context.Context, playerID uuid.UUID) ([]domain.SportsBetRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, player_id, event_id, market_id, selection_id,
		       stake_amount_minor, currency, odds_at_placement, potential_payout_minor,
		       status, payout_amount_minor, game_round_id, transaction_id, placed_at, settled_at
		FROM sports_bets WHERE player_id = $1
		ORDER BY placed_at DESC LIMIT 50`, playerID)
	if err != nil {
		return nil, domain.ErrInternal("query bets", err)
	}
	defer rows.Close()

	var bets []domain.SportsBetRecord
	for rows.Next() {
		var b domain.SportsBetRecord
		if err := rows.Scan(
			&b.ID, &b.PlayerID, &b.EventID, &b.MarketID, &b.SelectionID,
			&b.StakeAmountMinor, &b.Currency, &b.OddsAtPlacement, &b.PotentialPayoutMinor,
			&b.Status, &b.PayoutAmountMinor, &b.GameRoundID, &b.TransactionID, &b.PlacedAt, &b.SettledAt,
		); err != nil {
			return nil, domain.ErrInternal("scan bet", err)
		}
		bets = append(bets, b)
	}
	return bets, rows.Err()
}

// ListSports returns all active sports.
func (s *SportsbookService) ListSports(ctx context.Context, db repository.DBTX) ([]domain.Sport, error) {
	rows, err := db.Query(ctx,
		`SELECT id, key, name, icon, sort_order, active, created_at
		 FROM sports WHERE active = true ORDER BY sort_order ASC`)
	if err != nil {
		return nil, domain.ErrInternal("query sports", err)
	}
	defer rows.Close()

	var sports []domain.Sport
	for rows.Next() {
		var sp domain.Sport
		if err := rows.Scan(&sp.ID, &sp.Key, &sp.Name, &sp.Icon, &sp.SortOrder, &sp.Active, &sp.CreatedAt); err != nil {
			return nil, domain.ErrInternal("scan sport", err)
		}
		sports = append(sports, sp)
	}
	return sports, rows.Err()
}

// ListEvents returns events for a sport.
func (s *SportsbookService) ListEvents(ctx context.Context, db repository.DBTX, sportID uuid.UUID) ([]domain.SportsEvent, error) {
	rows, err := db.Query(ctx,
		`SELECT id, sport_id, league, home_team, away_team, start_time, status, score_home, score_away, created_at
		 FROM sports_events WHERE sport_id = $1 AND status IN ('upcoming', 'live')
		 ORDER BY start_time ASC`, sportID)
	if err != nil {
		return nil, domain.ErrInternal("query events", err)
	}
	defer rows.Close()

	var events []domain.SportsEvent
	for rows.Next() {
		var e domain.SportsEvent
		if err := rows.Scan(&e.ID, &e.SportID, &e.League, &e.HomeTeam, &e.AwayTeam, &e.StartTime, &e.Status, &e.ScoreHome, &e.ScoreAway, &e.CreatedAt); err != nil {
			return nil, domain.ErrInternal("scan event", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ListMarkets returns markets for an event.
func (s *SportsbookService) ListMarkets(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]domain.SportsMarket, error) {
	rows, err := db.Query(ctx,
		`SELECT id, event_id, name, type, status, specifiers, sort_order, created_at
		 FROM sports_markets WHERE event_id = $1 AND status = 'open'
		 ORDER BY sort_order ASC`, eventID)
	if err != nil {
		return nil, domain.ErrInternal("query markets", err)
	}
	defer rows.Close()

	var markets []domain.SportsMarket
	for rows.Next() {
		var m domain.SportsMarket
		if err := rows.Scan(&m.ID, &m.EventID, &m.Name, &m.Type, &m.Status, &m.Specifiers, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, domain.ErrInternal("scan market", err)
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

// ListSelections returns selections for a market.
func (s *SportsbookService) ListSelections(ctx context.Context, db repository.DBTX, marketID uuid.UUID) ([]domain.SportsSelection, error) {
	rows, err := db.Query(ctx,
		`SELECT id, market_id, name, odds_decimal, odds_fractional, odds_american, status, result, sort_order, created_at
		 FROM sports_selections WHERE market_id = $1 AND status = 'active'
		 ORDER BY sort_order ASC`, marketID)
	if err != nil {
		return nil, domain.ErrInternal("query selections", err)
	}
	defer rows.Close()

	var selections []domain.SportsSelection
	for rows.Next() {
		var s domain.SportsSelection
		if err := rows.Scan(&s.ID, &s.MarketID, &s.Name, &s.OddsDecimal, &s.OddsFractional, &s.OddsAmerican, &s.Status, &s.Result, &s.SortOrder, &s.CreatedAt); err != nil {
			return nil, domain.ErrInternal("scan selection", err)
		}
		selections = append(selections, s)
	}
	return selections, rows.Err()
}
