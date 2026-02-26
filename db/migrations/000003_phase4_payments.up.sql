CREATE TABLE IF NOT EXISTS payment_events (
  id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_id    uuid        NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
  status        varchar(20) NOT NULL,
  message       text,
  admin_user_id uuid,
  raw_data      jsonb       DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payment_events_payment_id_idx ON payment_events (payment_id);

ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider varchar(30);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider_session_id varchar(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider_payment_id varchar(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS approved_by uuid;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS approved_at timestamptz;
