CREATE TABLE IF NOT EXISTS sports (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  key        varchar(50) NOT NULL UNIQUE,
  name       varchar(100) NOT NULL,
  icon       varchar(50),
  sort_order integer     NOT NULL DEFAULT 0,
  active     boolean     NOT NULL DEFAULT true,
  created_at timestamp   DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sports_events (
  id         uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  sport_id   uuid         NOT NULL REFERENCES sports(id) ON DELETE CASCADE,
  league     varchar(200),
  home_team  varchar(200) NOT NULL,
  away_team  varchar(200) NOT NULL,
  start_time timestamptz  NOT NULL,
  status     varchar(30)  NOT NULL DEFAULT 'upcoming',
  score_home integer      DEFAULT 0,
  score_away integer      DEFAULT 0,
  metadata   jsonb        DEFAULT '{}',
  created_at timestamp    DEFAULT now(),
  updated_at timestamp    DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sports_events_sport_id_idx ON sports_events (sport_id);
CREATE INDEX IF NOT EXISTS sports_events_status_idx ON sports_events (status);
CREATE INDEX IF NOT EXISTS sports_events_start_time_idx ON sports_events (start_time);

CREATE TABLE IF NOT EXISTS sports_markets (
  id         uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id   uuid         NOT NULL REFERENCES sports_events(id) ON DELETE CASCADE,
  name       varchar(200) NOT NULL,
  type       varchar(50)  NOT NULL,
  status     varchar(30)  NOT NULL DEFAULT 'open',
  specifiers varchar(200),
  sort_order integer      NOT NULL DEFAULT 0,
  created_at timestamp    DEFAULT now(),
  updated_at timestamp    DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sports_markets_event_id_idx ON sports_markets (event_id);
CREATE INDEX IF NOT EXISTS sports_markets_status_idx ON sports_markets (status);

CREATE TABLE IF NOT EXISTS sports_selections (
  id              uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  market_id       uuid         NOT NULL REFERENCES sports_markets(id) ON DELETE CASCADE,
  name            varchar(200) NOT NULL,
  odds_decimal    integer      NOT NULL,
  odds_fractional varchar(20),
  odds_american   varchar(20),
  status          varchar(30)  NOT NULL DEFAULT 'active',
  result          varchar(20),
  sort_order      integer      NOT NULL DEFAULT 0,
  created_at      timestamp    DEFAULT now(),
  updated_at      timestamp    DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sports_selections_market_id_idx ON sports_selections (market_id);

CREATE TABLE IF NOT EXISTS sports_bets (
  id                   uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id            uuid         NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
  event_id             uuid         NOT NULL REFERENCES sports_events(id) ON DELETE CASCADE,
  market_id            uuid         NOT NULL REFERENCES sports_markets(id) ON DELETE CASCADE,
  selection_id         uuid         NOT NULL REFERENCES sports_selections(id) ON DELETE CASCADE,
  stake_amount_minor   integer      NOT NULL,
  currency             varchar(3)   NOT NULL DEFAULT 'EUR',
  odds_at_placement    integer      NOT NULL,
  potential_payout_minor integer    NOT NULL,
  status               varchar(30)  NOT NULL DEFAULT 'open',
  payout_amount_minor  integer      DEFAULT 0,
  game_round_id        varchar(200) NOT NULL,
  transaction_id       uuid,
  placed_at            timestamp    DEFAULT now(),
  settled_at           timestamp
);

CREATE INDEX IF NOT EXISTS sports_bets_player_id_status_idx ON sports_bets (player_id, status);
CREATE INDEX IF NOT EXISTS sports_bets_event_id_idx ON sports_bets (event_id);
CREATE INDEX IF NOT EXISTS sports_bets_game_round_id_idx ON sports_bets (game_round_id);
