ALTER TABLE apps
    DROP COLUMN IF EXISTS webhook_url,
    DROP COLUMN IF EXISTS webhook_secret;
