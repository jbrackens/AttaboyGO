-- 000006_phase6_gamification.up.sql
-- Gamification: quests, player progress, engagement tracking, reward grants

CREATE TABLE quests (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            varchar(200) NOT NULL,
    description     text,
    type            varchar(50)  NOT NULL DEFAULT 'standard',
    target_progress integer      NOT NULL DEFAULT 1,
    reward_amount   integer      NOT NULL,
    reward_currency varchar(3)   NOT NULL DEFAULT 'EUR',
    min_score       integer      NOT NULL DEFAULT 0,
    cooldown_minutes integer     NOT NULL DEFAULT 0,
    daily_budget_minor integer   NOT NULL DEFAULT 250000,
    active          boolean      NOT NULL DEFAULT true,
    sort_order      integer      NOT NULL DEFAULT 0,
    created_at      timestamp    DEFAULT now(),
    updated_at      timestamp    DEFAULT now()
);

CREATE TABLE player_quest_progress (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    quest_id        uuid        NOT NULL REFERENCES quests(id) ON DELETE CASCADE,
    progress        integer     NOT NULL DEFAULT 0,
    status          varchar(30) NOT NULL DEFAULT 'active',
    last_rewarded_at timestamp,
    completed_at    timestamp,
    claimed_at      timestamp,
    created_at      timestamp   DEFAULT now(),
    updated_at      timestamp   DEFAULT now(),
    UNIQUE (player_id, quest_id)
);

CREATE TABLE player_engagement (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id           uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    date                date        NOT NULL,
    video_minutes       integer     NOT NULL DEFAULT 0,
    social_interactions integer     NOT NULL DEFAULT 0,
    prediction_actions  integer     NOT NULL DEFAULT 0,
    wager_count         integer     NOT NULL DEFAULT 0,
    deposit_count       integer     NOT NULL DEFAULT 0,
    score               integer     NOT NULL DEFAULT 0,
    created_at          timestamp   DEFAULT now(),
    updated_at          timestamp   DEFAULT now(),
    UNIQUE (player_id, date)
);

CREATE TABLE reward_grants (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    quest_id    uuid        REFERENCES quests(id) ON DELETE SET NULL,
    reward_id   varchar(200),
    amount      integer     NOT NULL,
    currency    varchar(3)  NOT NULL DEFAULT 'EUR',
    granted_at  timestamp   DEFAULT now()
);

ALTER TABLE event_outbox ADD COLUMN "publishedAt" timestamp;

CREATE INDEX idx_event_outbox_unpublished ON event_outbox ("publishedAt") WHERE "publishedAt" IS NULL;
