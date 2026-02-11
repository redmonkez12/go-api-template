package user

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for user data persistence.
// Implementations exist for each supported database/ORM combination.
type RepositoryInterface interface {
	Create(ctx context.Context, email, passwordHash, verificationToken string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByVerificationToken(ctx context.Context, token string) (*User, error)
	CheckIfTokenAlreadyUsed(ctx context.Context, token string) (bool, error)
	MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token string) error
}
