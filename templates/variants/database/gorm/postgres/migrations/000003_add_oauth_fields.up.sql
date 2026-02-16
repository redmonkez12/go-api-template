ALTER TABLE users
    ALTER COLUMN password_hash DROP NOT NULL,
    ADD COLUMN auth_provider VARCHAR(20) NOT NULL DEFAULT 'local',
    ADD COLUMN provider_user_id VARCHAR(255);

CREATE UNIQUE INDEX idx_users_oauth_provider ON users(auth_provider, provider_user_id)
    WHERE provider_user_id IS NOT NULL;
