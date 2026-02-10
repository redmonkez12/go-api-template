package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const passwordResetTokenTTL = 1 * time.Hour

// PasswordResetRepository handles password reset token storage in Redis
type PasswordResetRepository struct {
	client *redis.Client
}

// NewPasswordResetRepository creates a new password reset repository instance
func NewPasswordResetRepository(client *redis.Client) *PasswordResetRepository {
	return &PasswordResetRepository{
		client: client,
	}
}

// StorePasswordResetToken stores a password reset token with 1-hour TTL
func (r *PasswordResetRepository) StorePasswordResetToken(ctx context.Context, userID uuid.UUID, token string) error {
	key := passwordResetKey(token)

	// Store user ID with TTL
	err := r.client.HSet(ctx, key, "user_id", userID.String()).Err()
	if err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}

	err = r.client.Expire(ctx, key, passwordResetTokenTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set TTL on password reset token: %w", err)
	}

	return nil
}

// GetPasswordResetToken retrieves the user ID associated with a password reset token
func (r *PasswordResetRepository) GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	key := passwordResetKey(token)

	userIDStr, err := r.client.HGet(ctx, key, "user_id").Result()
	if err == redis.Nil {
		return uuid.Nil, ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return userID, nil
}

// DeletePasswordResetToken removes a used password reset token
func (r *PasswordResetRepository) DeletePasswordResetToken(ctx context.Context, token string) error {
	key := passwordResetKey(token)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete password reset token: %w", err)
	}

	return nil
}

// passwordResetKey generates a Redis key for password reset tokens
func passwordResetKey(token string) string {
	// Hash the token for security
	hashedToken := hashToken(token)
	return fmt.Sprintf("password_reset:%s", hashedToken)
}
