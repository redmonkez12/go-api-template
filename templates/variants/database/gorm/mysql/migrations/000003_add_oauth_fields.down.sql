DROP INDEX idx_users_oauth_provider ON users;
ALTER TABLE users
    DROP COLUMN provider_user_id,
    DROP COLUMN auth_provider,
    MODIFY password_hash VARCHAR(255) NOT NULL;
