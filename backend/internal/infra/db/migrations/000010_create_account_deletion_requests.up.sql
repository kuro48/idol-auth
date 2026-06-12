CREATE TABLE IF NOT EXISTS account_deletion_requests (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_id    TEXT NOT NULL UNIQUE,
    status         TEXT NOT NULL CHECK (status IN ('scheduled', 'cancelled', 'completed')),
    reason         TEXT,
    requested_at   TIMESTAMPTZ NOT NULL,
    scheduled_for  TIMESTAMPTZ NOT NULL,
    cancelled_at   TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    last_actor_id  TEXT
);

CREATE INDEX IF NOT EXISTS idx_account_deletion_requests_due
    ON account_deletion_requests (status, scheduled_for);
