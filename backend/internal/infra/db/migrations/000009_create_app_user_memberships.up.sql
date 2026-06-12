CREATE TABLE IF NOT EXISTS app_user_memberships (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    identity_id  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    profile      JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   TEXT,
    updated_by   TEXT,
    CONSTRAINT app_user_memberships_app_identity_unique UNIQUE (app_id, identity_id)
);

CREATE INDEX IF NOT EXISTS idx_app_user_memberships_identity_id ON app_user_memberships (identity_id);
CREATE INDEX IF NOT EXISTS idx_app_user_memberships_app_status ON app_user_memberships (app_id, status);
