CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS admin_users (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  email         citext      UNIQUE NOT NULL,
  password_hash varchar(128) NOT NULL,
  display_name  varchar(100) NOT NULL,
  role          varchar(20) NOT NULL DEFAULT 'admin',
  active        boolean     DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS player_notes (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id     uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
  admin_user_id uuid        NOT NULL REFERENCES admin_users(id),
  content       text        NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS player_notes_player_id_idx ON player_notes (player_id);
