ALTER TABLE app_management_tokens
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS last_used_at;
