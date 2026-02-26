-- 000010_phase11_dome_predictions.up.sql
-- Dome prediction market integrations (Polymarket, Kalshi)

ALTER TABLE prediction_markets
    ADD COLUMN dome_platform varchar(20),
    ADD COLUMN dome_market_slug varchar(300),
    ADD COLUMN dome_condition_id varchar(200),
    ADD COLUMN dome_event_slug varchar(300),
    ADD COLUMN dome_metadata jsonb DEFAULT '{}',
    ADD COLUMN dome_last_price_at bigint,
    ADD COLUMN dome_auto_settle boolean NOT NULL DEFAULT true,
    ADD COLUMN tags jsonb DEFAULT '[]';

CREATE TABLE dome_feed_state (
    feed_name       varchar(50) PRIMARY KEY,
    last_sync_at    timestamp,
    last_sync_count integer     NOT NULL DEFAULT 0,
    status          varchar(30) NOT NULL DEFAULT 'idle',
    error_message   text,
    metadata        jsonb       DEFAULT '{}',
    created_at      timestamptz DEFAULT now(),
    updated_at      timestamptz DEFAULT now()
);

INSERT INTO dome_feed_state (feed_name, status)
VALUES
    ('polymarket-sync', 'idle'),
    ('kalshi-sync', 'idle'),
    ('price-updater', 'idle'),
    ('settlement-checker', 'idle');

CREATE UNIQUE INDEX idx_prediction_markets_dome_slug
    ON prediction_markets (dome_platform, dome_market_slug)
    WHERE dome_market_slug IS NOT NULL;

CREATE INDEX idx_prediction_markets_dome_platform
    ON prediction_markets (dome_platform)
    WHERE dome_platform IS NOT NULL;
