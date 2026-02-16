DROP INDEX IF EXISTS idx_users_oauth_provider;
ALTER TABLE users
    DROP COLUMN IF EXISTS provider_user_id,
    DROP COLUMN IF EXISTS auth_provider,
    ALTER COLUMN password_hash SET NOT NULL;
