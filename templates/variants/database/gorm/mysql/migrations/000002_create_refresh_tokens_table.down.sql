DROP INDEX idx_refresh_tokens_user_id ON refresh_tokens;
DROP INDEX idx_refresh_tokens_token_hash ON refresh_tokens;
DROP TABLE IF EXISTS refresh_tokens;
