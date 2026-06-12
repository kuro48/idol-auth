CREATE TABLE IF NOT EXISTS oidc_client_scopes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_client_id UUID NOT NULL REFERENCES oidc_clients (id) ON DELETE CASCADE,
    scope          TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT oidc_client_scopes_unique UNIQUE (oidc_client_id, scope)
);
