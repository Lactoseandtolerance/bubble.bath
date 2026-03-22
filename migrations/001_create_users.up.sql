CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    digit_code      SMALLINT NOT NULL CHECK (digit_code >= 0 AND digit_code <= 99),
    hue_encrypted   BYTEA NOT NULL,
    sat_encrypted   BYTEA NOT NULL,
    val_encrypted   BYTEA NOT NULL,
    color_hash      BYTEA NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    avatar_shape    TEXT NOT NULL DEFAULT '',
    recovery_secret BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_digit_code ON users (digit_code);
