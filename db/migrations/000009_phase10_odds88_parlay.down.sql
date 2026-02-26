-- 000009_phase10_odds88_parlay.down.sql

DROP TABLE IF EXISTS sports_parlay_legs;
DROP TABLE IF EXISTS sports_parlay_bets;
DROP TABLE IF EXISTS odds88_bet_map;
DROP TABLE IF EXISTS odds88_player_map;
DROP TABLE IF EXISTS odds88_feed_state;

DROP INDEX IF EXISTS idx_sports_selections_odds88;
DROP INDEX IF EXISTS idx_sports_markets_odds88;
DROP INDEX IF EXISTS idx_sports_events_odds88;

ALTER TABLE sports_selections DROP COLUMN odds88_selection_id;
ALTER TABLE sports_markets DROP COLUMN odds88_market_id;
ALTER TABLE sports_events DROP COLUMN odds88_event_id, DROP COLUMN odds88_trading_status;
ALTER TABLE sports DROP COLUMN odds88_sport_id;
