CREATE TABLE IF NOT EXISTS secrets (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT          NOT NULL,
    data       BYTEA         NOT NULL,
    meta       TEXT          NOT NULL DEFAULT '',
    revision   INTEGER       NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets (user_id);
CREATE INDEX IF NOT EXISTS idx_secrets_updated_at ON secrets (updated_at);