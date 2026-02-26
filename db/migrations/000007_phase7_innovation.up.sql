-- 000007_phase7_innovation.up.sql
-- Innovation layer: plugins, predictions, AI conversations, video, social

CREATE TABLE plugins (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id         varchar(200) NOT NULL UNIQUE,
    name              varchar(300) NOT NULL,
    description       text,
    domain            varchar(50)  NOT NULL,
    scopes            jsonb        NOT NULL DEFAULT '[]',
    rate_limit        integer      NOT NULL DEFAULT 60,
    risk_tier         varchar(20)  NOT NULL DEFAULT 'standard',
    active            boolean      NOT NULL DEFAULT true,
    requires_approval boolean      NOT NULL DEFAULT false,
    created_at        timestamp    DEFAULT now(),
    updated_at        timestamp    DEFAULT now()
);

CREATE TABLE plugin_dispatches (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id       varchar(200) NOT NULL,
    scope           varchar(100) NOT NULL,
    payload         jsonb,
    status          varchar(30)  NOT NULL DEFAULT 'pending',
    result          jsonb,
    error           text,
    player_id       uuid         REFERENCES v2_players(id) ON DELETE SET NULL,
    idempotency_key varchar(200) UNIQUE,
    created_at      timestamp    DEFAULT now(),
    updated_at      timestamp    DEFAULT now()
);

CREATE INDEX idx_plugin_dispatches_plugin_id ON plugin_dispatches (plugin_id);

CREATE TABLE prediction_markets (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    title               varchar(300) NOT NULL,
    description         text,
    category            varchar(100) NOT NULL DEFAULT 'general',
    status              varchar(30)  NOT NULL DEFAULT 'open',
    close_at            timestamp,
    outcomes            jsonb        NOT NULL DEFAULT '[]',
    winning_outcome_id  uuid,
    attestation         jsonb,
    created_by          uuid         NOT NULL REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at          timestamp    DEFAULT now(),
    updated_at          timestamp    DEFAULT now()
);

CREATE TABLE prediction_stakes (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id           uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    market_id           uuid        NOT NULL REFERENCES prediction_markets(id) ON DELETE CASCADE,
    outcome_id          varchar(100) NOT NULL,
    stake_amount_minor  integer      NOT NULL,
    currency            varchar(3)   NOT NULL DEFAULT 'EUR',
    status              varchar(30)  NOT NULL DEFAULT 'active',
    payout_amount_minor integer      DEFAULT 0,
    placed_at           timestamp    DEFAULT now(),
    settled_at          timestamp
);

CREATE INDEX idx_prediction_stakes_player_market ON prediction_stakes (player_id, market_id);

CREATE TABLE ai_conversations (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    created_at  timestamp   DEFAULT now(),
    updated_at  timestamp   DEFAULT now()
);

CREATE INDEX idx_ai_conversations_player_id ON ai_conversations (player_id);

CREATE TABLE ai_messages (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid        NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            varchar(20)  NOT NULL,
    content         text         NOT NULL,
    model           varchar(100),
    tokens_used     integer      DEFAULT 0,
    blocked         boolean      NOT NULL DEFAULT false,
    blocked_reason  varchar(200),
    created_at      timestamp    DEFAULT now()
);

CREATE INDEX idx_ai_messages_conversation_id ON ai_messages (conversation_id);

CREATE TABLE video_sessions (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id        uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    stream_url       varchar(500),
    status           varchar(30) NOT NULL DEFAULT 'active',
    started_at       timestamp   DEFAULT now(),
    ended_at         timestamp,
    duration_minutes integer     DEFAULT 0
);

CREATE INDEX idx_video_sessions_player_id ON video_sessions (player_id);

CREATE TABLE social_posts (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    content     text        NOT NULL,
    type        varchar(50) NOT NULL DEFAULT 'text',
    target_type varchar(50),
    target_id   uuid,
    created_at  timestamp   DEFAULT now(),
    updated_at  timestamp   DEFAULT now()
);

CREATE INDEX idx_social_posts_player_id ON social_posts (player_id);
