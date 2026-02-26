-- 000010_phase11_dome_predictions.down.sql

DROP INDEX IF EXISTS idx_prediction_markets_dome_platform;
DROP INDEX IF EXISTS idx_prediction_markets_dome_slug;

DROP TABLE IF EXISTS dome_feed_state;

ALTER TABLE prediction_markets
    DROP COLUMN tags,
    DROP COLUMN dome_auto_settle,
    DROP COLUMN dome_last_price_at,
    DROP COLUMN dome_metadata,
    DROP COLUMN dome_event_slug,
    DROP COLUMN dome_condition_id,
    DROP COLUMN dome_market_slug,
    DROP COLUMN dome_platform;
