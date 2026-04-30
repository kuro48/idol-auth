CREATE TABLE IF NOT EXISTS app_management_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'rotated')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   TEXT,
    updated_by   TEXT
);

CREATE INDEX IF NOT EXISTS idx_app_management_tokens_app_id ON app_management_tokens (app_id);
