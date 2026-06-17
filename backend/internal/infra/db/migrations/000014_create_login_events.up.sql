CREATE TABLE IF NOT EXISTS login_events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_id      TEXT NOT NULL,
    session_id       TEXT NOT NULL,
    authenticated_at TIMESTAMPTZ NOT NULL,
    issued_at        TIMESTAMPTZ NOT NULL,
    aal              TEXT NOT NULL DEFAULT '',
    methods          TEXT[] NOT NULL DEFAULT '{}',
    ip_address       INET,
    user_agent       TEXT,
    recorded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id)
);

CREATE INDEX IF NOT EXISTS idx_login_events_identity_id      ON login_events (identity_id);
CREATE INDEX IF NOT EXISTS idx_login_events_authenticated_at ON login_events (authenticated_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_events_identity_at      ON login_events (identity_id, authenticated_at DESC);
