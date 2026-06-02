ALTER TABLE apps
    ADD COLUMN webhook_url    TEXT,
    ADD COLUMN webhook_secret TEXT;
