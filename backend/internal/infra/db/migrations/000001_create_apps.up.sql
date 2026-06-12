CREATE TABLE IF NOT EXISTS apps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('web', 'spa', 'native', 'm2m')),
    party_type  TEXT NOT NULL CHECK (party_type IN ('first_party', 'third_party')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  TEXT,
    updated_by  TEXT,
    CONSTRAINT apps_slug_unique UNIQUE (slug)
);
