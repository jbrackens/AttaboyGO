-- 000009_phase10_odds88_parlay.up.sql
-- Odds88 integration and parlay betting

ALTER TABLE sports ADD COLUMN odds88_sport_id integer UNIQUE;

ALTER TABLE sports_events
    ADD COLUMN odds88_event_id bigint UNIQUE,
    ADD COLUMN odds88_trading_status varchar(20) DEFAULT 'open';

ALTER TABLE sports_markets ADD COLUMN odds88_market_id varchar(200) UNIQUE;

ALTER TABLE sports_selections ADD COLUMN odds88_selection_id bigint UNIQUE;

CREATE TABLE odds88_feed_state (
    feed_name         varchar(50) PRIMARY KEY,
    last_revision     bigint      NOT NULL DEFAULT 0,
    last_updated_at   timestamp   DEFAULT now(),
    connection_status varchar(30) NOT NULL DEFAULT 'disconnected',
    error_message     text
);

INSERT INTO odds88_feed_state (feed_name, last_revision, connection_status)
VALUES
    ('delta', 0, 'disconnected'),
    ('settlement', 0, 'disconnected');

CREATE TABLE odds88_player_map (
    player_id        uuid        PRIMARY KEY REFERENCES v2_players(id) ON DELETE CASCADE,
    odds88_player_id varchar(100) NOT NULL UNIQUE,
    created_at       timestamp   DEFAULT now()
);

CREATE TABLE odds88_bet_map (
    bet_id        uuid        PRIMARY KEY,
    odds88_bet_id varchar(100) NOT NULL UNIQUE,
    odds88_status varchar(30),
    created_at    timestamp   DEFAULT now()
);

CREATE TABLE sports_parlay_bets (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id            uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    stake_amount_minor   integer     NOT NULL,
    currency             varchar(3)  NOT NULL DEFAULT 'EUR',
    combined_odds_decimal integer    NOT NULL,
    potential_payout_minor integer   NOT NULL,
    status               varchar(30) NOT NULL DEFAULT 'open',
    payout_amount_minor  integer     DEFAULT 0,
    game_round_id        varchar(200) NOT NULL,
    transaction_id       uuid,
    num_legs             integer     NOT NULL,
    num_legs_won         integer     NOT NULL DEFAULT 0,
    num_legs_lost        integer     NOT NULL DEFAULT 0,
    num_legs_voided      integer     NOT NULL DEFAULT 0,
    num_legs_open        integer     NOT NULL,
    placed_at            timestamp   DEFAULT now(),
    settled_at           timestamp
);

CREATE INDEX idx_parlay_bets_player_status ON sports_parlay_bets (player_id, status);

CREATE TABLE sports_parlay_legs (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    parlay_bet_id     uuid        NOT NULL REFERENCES sports_parlay_bets(id) ON DELETE CASCADE,
    event_id          uuid        NOT NULL REFERENCES sports_events(id) ON DELETE CASCADE,
    market_id         uuid        NOT NULL REFERENCES sports_markets(id) ON DELETE CASCADE,
    selection_id      uuid        NOT NULL REFERENCES sports_selections(id) ON DELETE CASCADE,
    odds_at_placement integer     NOT NULL,
    status            varchar(30) NOT NULL DEFAULT 'open',
    settled_at        timestamp,
    sort_order        integer     NOT NULL DEFAULT 0
);

CREATE INDEX idx_parlay_legs_parlay ON sports_parlay_legs (parlay_bet_id);
CREATE INDEX idx_parlay_legs_selection ON sports_parlay_legs (selection_id);

CREATE INDEX idx_sports_events_odds88 ON sports_events (odds88_event_id) WHERE odds88_event_id IS NOT NULL;
CREATE INDEX idx_sports_markets_odds88 ON sports_markets (odds88_market_id) WHERE odds88_market_id IS NOT NULL;
CREATE INDEX idx_sports_selections_odds88 ON sports_selections (odds88_selection_id) WHERE odds88_selection_id IS NOT NULL;
