CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS v2_players (
  id              uuid          PRIMARY KEY,
  balance         numeric(15,0) NOT NULL DEFAULT 0,
  bonus_balance   numeric(15,0) NOT NULL DEFAULT 0,
  reserved_balance numeric(15,0) NOT NULL DEFAULT 0,
  currency        varchar(3)    NOT NULL,
  created_at      timestamptz   NOT NULL DEFAULT now(),
  updated_at      timestamptz   NOT NULL DEFAULT now(),
  CONSTRAINT v2_players_balance_non_negative CHECK (balance >= 0),
  CONSTRAINT v2_players_bonus_balance_non_negative CHECK (bonus_balance >= 0),
  CONSTRAINT v2_players_reserved_balance_non_negative CHECK (reserved_balance >= 0)
);

CREATE TABLE IF NOT EXISTS v2_transactions (
  id                      uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id               uuid          NOT NULL REFERENCES v2_players(id),
  type                    varchar(40)   NOT NULL,
  amount                  numeric(15,0) NOT NULL,
  balance_after           numeric(15,0) NOT NULL,
  bonus_balance_after     numeric(15,0) NOT NULL,
  reserved_balance_after  numeric(15,0) NOT NULL,
  external_transaction_id varchar(128),
  manufacturer_id         varchar(64),
  sub_transaction_id      varchar(64),
  target_transaction_id   uuid          REFERENCES v2_transactions(id),
  game_round_id           varchar(128),
  metadata                jsonb         NOT NULL DEFAULT '{}',
  created_at              timestamptz   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS v2_transactions_idempotency_idx
  ON v2_transactions (player_id, manufacturer_id, external_transaction_id, sub_transaction_id)
  WHERE external_transaction_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS v2_transactions_player_created_idx
  ON v2_transactions (player_id, created_at);

CREATE INDEX IF NOT EXISTS v2_transactions_game_round_idx
  ON v2_transactions (game_round_id)
  WHERE game_round_id IS NOT NULL;

-- Event outbox uses quoted camelCase columns (Audit #6: mixed naming convention)
CREATE TABLE IF NOT EXISTS event_outbox (
  "id" bigserial PRIMARY KEY,
  "eventId" uuid NOT NULL DEFAULT gen_random_uuid(),
  "aggregateType" varchar(64) NOT NULL,
  "aggregateId" varchar(128) NOT NULL,
  "eventType" varchar(128) NOT NULL,
  "partitionKey" varchar(128),
  "headers" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "payload" jsonb NOT NULL,
  "occurredAt" timestamptz NOT NULL DEFAULT now(),
  "createdAt" timestamptz NOT NULL DEFAULT now(),
  UNIQUE ("eventId")
);
