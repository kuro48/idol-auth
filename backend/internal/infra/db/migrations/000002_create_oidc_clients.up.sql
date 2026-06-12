CREATE TABLE IF NOT EXISTS oidc_clients (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hydra_client_id               TEXT NOT NULL,
    app_id                        UUID NOT NULL REFERENCES apps (id),
    client_type                   TEXT NOT NULL CHECK (client_type IN ('public', 'confidential')),
    status                        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'rotated')),
    token_endpoint_auth_method    TEXT NOT NULL,
    pkce_required                 BOOLEAN NOT NULL DEFAULT TRUE,
    redirect_uri_mode             TEXT NOT NULL DEFAULT 'strict',
    post_logout_redirect_uri_mode TEXT NOT NULL DEFAULT 'strict',
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                    TEXT,
    updated_by                    TEXT,
    CONSTRAINT oidc_clients_hydra_client_id_unique UNIQUE (hydra_client_id)
);
