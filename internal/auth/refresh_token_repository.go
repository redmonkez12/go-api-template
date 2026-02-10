package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RefreshTokenRepository defines the interface for refresh token storage
type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) error
}
