CREATE TABLE IF NOT EXISTS login_attempts (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL,
    realm         TEXT NOT NULL DEFAULT 'player',
    ip_address    TEXT NOT NULL DEFAULT '',
    success       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_login_attempts_email_realm ON login_attempts (email, realm, created_at);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    realm         TEXT NOT NULL DEFAULT 'player',
    token_hash    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_prt_email_realm ON password_reset_tokens (email, realm);
CREATE INDEX idx_prt_token_hash ON password_reset_tokens (token_hash);
