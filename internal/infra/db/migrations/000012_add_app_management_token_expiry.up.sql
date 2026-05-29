ALTER TABLE app_management_tokens
    ADD COLUMN IF NOT EXISTS expires_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

-- Back-fill: all active tokens get a 1-year TTL from now.
UPDATE app_management_tokens
SET expires_at = NOW() + INTERVAL '1 year'
WHERE expires_at IS NULL;

-- Enforce NOT NULL now that every row has a value.
ALTER TABLE app_management_tokens
    ALTER COLUMN expires_at SET NOT NULL;
