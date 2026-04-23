CREATE TABLE IF NOT EXISTS oidc_client_redirect_uris (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_client_id UUID NOT NULL REFERENCES oidc_clients (id) ON DELETE CASCADE,
    uri            TEXT NOT NULL,
    kind           TEXT NOT NULL CHECK (kind IN ('login_callback', 'post_logout_callback')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT oidc_client_redirect_uris_unique UNIQUE (oidc_client_id, uri, kind)
);
