package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"go-api-template/internal/database"
)

// Repository handles refresh token persistence
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// StoreRefreshToken stores a refresh token in the database
func (r *Repository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	tokenHash := hashToken(token)

	dbToken := &database.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	_, err := r.db.NewInsert().
		Model(dbToken).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by its hash
func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	tokenHash := hashToken(token)

	dbToken := new(database.RefreshToken)
	err := r.db.NewSelect().
		Model(dbToken).
		Where("token_hash = ?", tokenHash).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return mapDBRefreshTokenToModel(dbToken), nil
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *Repository) RevokeRefreshToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)

	result, err := r.db.NewUpdate().
		Model((*database.RefreshToken)(nil)).
		Set("revoked_at = NOW()").
		Where("token_hash = ?", tokenHash).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*database.RefreshToken)(nil)).
		Set("revoked_at = NOW()").
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens from the database
// Should be run periodically (e.g., via cron job)
func (r *Repository) CleanupExpiredTokens(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*database.RefreshToken)(nil)).
		Where("expires_at < NOW()").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return nil
}

// mapDBRefreshTokenToModel converts database model to domain model
func mapDBRefreshTokenToModel(dbt *database.RefreshToken) *RefreshToken {
	return &RefreshToken{
		ID:        dbt.ID,
		UserID:    dbt.UserID,
		TokenHash: dbt.TokenHash,
		ExpiresAt: dbt.ExpiresAt,
		CreatedAt: dbt.CreatedAt,
		RevokedAt: dbt.RevokedAt,
	}
}
