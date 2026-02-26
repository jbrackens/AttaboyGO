ALTER TABLE payments DROP COLUMN IF EXISTS provider;
ALTER TABLE payments DROP COLUMN IF EXISTS provider_session_id;
ALTER TABLE payments DROP COLUMN IF EXISTS provider_payment_id;
ALTER TABLE payments DROP COLUMN IF EXISTS approved_by;
ALTER TABLE payments DROP COLUMN IF EXISTS approved_at;

DROP TABLE IF EXISTS payment_events CASCADE;
