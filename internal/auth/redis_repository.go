package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisRepository handles refresh token persistence in Redis
type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// getTokenKey generates the Redis key for a refresh token
func getTokenKey(tokenHash string) string {
	return fmt.Sprintf("refresh_token:%s", tokenHash)
}

// getRevokedKey generates the Redis key for a revoked token marker
func getRevokedKey(tokenHash string) string {
	return fmt.Sprintf("refresh_token:revoked:%s", tokenHash)
}

// getUserTokensKey generates the Redis key for user's token set
func getUserTokensKey(userID uuid.UUID) string {
	return fmt.Sprintf("user_tokens:%s", userID.String())
}

// StoreRefreshToken stores a refresh token in Redis with TTL
func (r *RedisRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	tokenHash := hashToken(token)
	tokenKey := getTokenKey(tokenHash)
	userTokensKey := getUserTokensKey(userID)

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return fmt.Errorf("token expiration time is in the past")
	}

	// Create a pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Store token with user_id and expiration as a hash
	pipe.HSet(ctx, tokenKey, map[string]interface{}{
		"user_id":    userID.String(),
		"expires_at": expiresAt.Unix(),
		"created_at": time.Now().Unix(),
	})
	pipe.Expire(ctx, tokenKey, ttl)

	// Add token hash to user's set of tokens (also with TTL)
	pipe.SAdd(ctx, userTokensKey, tokenHash)
	pipe.Expire(ctx, userTokensKey, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by its hash
func (r *RedisRepository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	tokenHash := hashToken(token)
	tokenKey := getTokenKey(tokenHash)
	revokedKey := getRevokedKey(tokenHash)

	// Check if token is revoked
	revoked, err := r.client.Exists(ctx, revokedKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check revocation: %w", err)
	}
	if revoked > 0 {
		return nil, ErrRefreshTokenRevoked
	}

	// Get token data
	data, err := r.client.HGetAll(ctx, tokenKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if len(data) == 0 {
		return nil, ErrRefreshTokenNotFound
	}

	// Parse user_id
	userID, err := uuid.Parse(data["user_id"])
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Parse expires_at
	var expiresAtUnix int64
	fmt.Sscanf(data["expires_at"], "%d", &expiresAtUnix)
	expiresAt := time.Unix(expiresAtUnix, 0)

	// Check expiration
	if time.Now().After(expiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	// Parse created_at
	var createdAtUnix int64
	fmt.Sscanf(data["created_at"], "%d", &createdAtUnix)
	createdAt := time.Unix(createdAtUnix, 0)

	return &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
		RevokedAt: nil, // Not revoked if we got here
	}, nil
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *RedisRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	tokenKey := getTokenKey(tokenHash)
	revokedKey := getRevokedKey(tokenHash)

	// Check if token exists
	exists, err := r.client.Exists(ctx, tokenKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check token existence: %w", err)
	}
	if exists == 0 {
		return ErrRefreshTokenNotFound
	}

	// Get TTL from original token
	ttl, err := r.client.TTL(ctx, tokenKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get token TTL: %w", err)
	}

	// Mark as revoked with same TTL as the token
	if ttl > 0 {
		err = r.client.Set(ctx, revokedKey, "1", ttl).Err()
	} else {
		// Fallback if TTL is not available
		err = r.client.Set(ctx, revokedKey, "1", 7*24*time.Hour).Err()
	}

	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RedisRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	userTokensKey := getUserTokensKey(userID)

	// Get all token hashes for this user
	tokenHashes, err := r.client.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil // No tokens to revoke
	}

	// Revoke each token
	pipe := r.client.Pipeline()
	for _, tokenHash := range tokenHashes {
		tokenKey := getTokenKey(tokenHash)
		revokedKey := getRevokedKey(tokenHash)

		// Get TTL from original token
		ttl, _ := r.client.TTL(ctx, tokenKey).Result()
		if ttl > 0 {
			pipe.Set(ctx, revokedKey, "1", ttl)
		} else {
			pipe.Set(ctx, revokedKey, "1", 7*24*time.Hour)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens is not needed for Redis as TTL handles expiration automatically
// This method is kept for interface compatibility but does nothing
func (r *RedisRepository) CleanupExpiredTokens(ctx context.Context) error {
	// Redis handles expiration automatically via TTL
	// No manual cleanup needed
	return nil
}
