CREATE TABLE IF NOT EXISTS app_registration_requests (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_id                 TEXT NOT NULL,
    status                      TEXT NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending','under_review','approved','rejected','withdrawn','changes_requested')),
    name                        TEXT NOT NULL,
    slug                        TEXT NOT NULL,
    type                        TEXT NOT NULL CHECK (type IN ('web','spa','native','m2m')),
    description                 TEXT NOT NULL,
    homepage_url                TEXT NOT NULL DEFAULT '',
    privacy_policy_url          TEXT NOT NULL DEFAULT '',
    terms_url                   TEXT NOT NULL DEFAULT '',
    contact_email               TEXT NOT NULL,
    organization                TEXT NOT NULL DEFAULT '',
    purpose                     TEXT NOT NULL,
    redirect_uris               TEXT[] NOT NULL DEFAULT '{}',
    post_logout_redirect_uris   TEXT[] NOT NULL DEFAULT '{}',
    scopes                      TEXT[] NOT NULL DEFAULT '{}',
    reviewer_id                 TEXT,
    reviewer_note               TEXT,
    decided_at                  TIMESTAMPTZ,
    created_app_id              UUID REFERENCES apps(id) ON DELETE SET NULL,
    created_client_id           UUID REFERENCES oidc_clients(id) ON DELETE SET NULL,
    version                     INT NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_app_registration_requests_identity
    ON app_registration_requests (identity_id);

CREATE INDEX IF NOT EXISTS idx_app_registration_requests_status
    ON app_registration_requests (status);

CREATE INDEX IF NOT EXISTS idx_app_registration_requests_identity_status
    ON app_registration_requests (identity_id, status);

CREATE TABLE IF NOT EXISTS app_registration_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id  UUID NOT NULL REFERENCES app_registration_requests(id) ON DELETE CASCADE,
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('developer','admin','system')),
    actor_id    TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    note        TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_app_registration_events_request
    ON app_registration_events (request_id, occurred_at);
