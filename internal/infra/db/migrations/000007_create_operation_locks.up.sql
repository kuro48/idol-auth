CREATE TABLE IF NOT EXISTS operation_locks (
    key        TEXT PRIMARY KEY,
    owner      TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
