CREATE TABLE IF NOT EXISTS admin_clients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hydra_client_id TEXT NOT NULL,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_clients_hydra_client_id_unique UNIQUE (hydra_client_id)
);
