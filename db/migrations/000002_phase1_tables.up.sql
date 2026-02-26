CREATE TABLE IF NOT EXISTS auth_users (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  email         citext      UNIQUE NOT NULL,
  password_hash varchar(128) NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS player_profiles (
  player_id      uuid        PRIMARY KEY REFERENCES v2_players(id) ON DELETE CASCADE,
  email          citext      UNIQUE NOT NULL,
  first_name     varchar(100),
  last_name      varchar(100),
  date_of_birth  date,
  country        varchar(2),
  currency       varchar(3)  NOT NULL,
  language       varchar(2)  DEFAULT 'en',
  mobile_phone   varchar(20),
  address        text,
  post_code      varchar(20),
  verified       boolean     DEFAULT false,
  account_status varchar(20) DEFAULT 'active',
  risk_profile   varchar(10) DEFAULT 'low',
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  id         uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id  uuid         REFERENCES v2_players(id) ON DELETE CASCADE,
  token      varchar(512) UNIQUE NOT NULL,
  ip_address inet,
  user_agent text,
  created_at timestamptz  NOT NULL DEFAULT now(),
  expires_at timestamptz  NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_player_id_idx ON sessions (player_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS game_manufacturers (
  id     varchar(4)   PRIMARY KEY,
  name   varchar(100) NOT NULL,
  active boolean      DEFAULT true
);

CREATE TABLE IF NOT EXISTS games (
  id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  manufacturer_id  varchar(4)   REFERENCES game_manufacturers(id),
  external_game_id varchar(200),
  name             varchar(200) NOT NULL,
  category         varchar(50),
  rtp              decimal(5,2),
  mobile           boolean      DEFAULT true,
  demo_available   boolean      DEFAULT true,
  active           boolean      DEFAULT true,
  metadata         jsonb        DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS games_manufacturer_id_idx ON games (manufacturer_id);
CREATE INDEX IF NOT EXISTS games_category_idx ON games (category);

CREATE TABLE IF NOT EXISTS payment_methods (
  id     uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  name   varchar(100) NOT NULL,
  type   varchar(50),
  active boolean      DEFAULT true
);

CREATE TABLE IF NOT EXISTS payments (
  id                      uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id               uuid         REFERENCES v2_players(id) ON DELETE CASCADE,
  type                    varchar(10)  NOT NULL,
  amount                  decimal(15,0) NOT NULL,
  currency                varchar(3)   NOT NULL,
  status                  varchar(20)  NOT NULL,
  payment_method_id       uuid         REFERENCES payment_methods(id),
  external_transaction_id varchar(200),
  transaction_id          uuid         REFERENCES v2_transactions(id),
  metadata                jsonb        DEFAULT '{}',
  created_at              timestamptz  NOT NULL DEFAULT now(),
  updated_at              timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payments_player_id_idx ON payments (player_id);
CREATE INDEX IF NOT EXISTS payments_status_idx ON payments (status);

CREATE TABLE IF NOT EXISTS player_limits (
  id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id   uuid         REFERENCES v2_players(id) ON DELETE CASCADE,
  type        varchar(30)  NOT NULL,
  period      varchar(10),
  limit_value decimal(15,0),
  active      boolean      DEFAULT true,
  permanent   boolean      DEFAULT false,
  expires_at  timestamptz,
  created_at  timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS player_limits_player_id_idx ON player_limits (player_id);

CREATE TABLE IF NOT EXISTS bonuses (
  id                  uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  name                varchar(200),
  code                varchar(50)  UNIQUE,
  wagering_multiplier decimal(5,1),
  min_deposit         decimal(15,0),
  max_bonus           decimal(15,0),
  days_until_expiry   integer      DEFAULT 30,
  active              boolean      DEFAULT true
);

CREATE TABLE IF NOT EXISTS player_bonuses (
  id                  uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id           uuid         REFERENCES v2_players(id) ON DELETE CASCADE,
  bonus_id            uuid         REFERENCES bonuses(id),
  status              varchar(20)  DEFAULT 'active',
  initial_amount      decimal(15,0),
  wagering_requirement decimal(15,0),
  wagered             decimal(15,0) DEFAULT 0,
  expires_at          timestamptz,
  created_at          timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS player_bonuses_player_id_idx ON player_bonuses (player_id);
CREATE INDEX IF NOT EXISTS player_bonuses_status_idx ON player_bonuses (status);
