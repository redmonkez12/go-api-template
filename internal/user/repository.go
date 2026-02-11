package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/redmonkez12/go-api-template/internal/database"
)

var (
	ErrNotFound       = errors.New("user not found")
	ErrDuplicateEmail = errors.New("email already exists")
)

// Repository handles user data persistence
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new user into the database
func (r *Repository) Create(ctx context.Context, email, passwordHash, verificationToken string) (*User, error) {
	now := time.Now()
	dbUser := &database.User{
		Email:                     email,
		PasswordHash:              passwordHash,
		EmailVerificationToken:    &verificationToken,
		EmailVerificationSentAt:   &now,
		EmailVerified:             false,
	}

	_, err := r.db.NewInsert().
		Model(dbUser).
		Returning("*").
		Exec(ctx)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return mapDBUserToModel(dbUser), nil
}

// GetByEmail retrieves a user by email
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	dbUser := new(database.User)
	err := r.db.NewSelect().
		Model(dbUser).
		Where("email = ?", email).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return mapDBUserToModel(dbUser), nil
}

// GetByID retrieves a user by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	dbUser := new(database.User)
	err := r.db.NewSelect().
		Model(dbUser).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return mapDBUserToModel(dbUser), nil
}

// GetByVerificationToken retrieves a user by verification token
func (r *Repository) GetByVerificationToken(ctx context.Context, token string) (*User, error) {
	dbUser := new(database.User)
	err := r.db.NewSelect().
		Model(dbUser).
		Where("email_verification_token = ?", token).
		Where("email_verified = ?", false).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by verification token: %w", err)
	}

	return mapDBUserToModel(dbUser), nil
}

// CheckIfTokenAlreadyUsed checks if a verification token was already used (email verified)
func (r *Repository) CheckIfTokenAlreadyUsed(ctx context.Context, token string) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*database.User)(nil)).
		Where("email_verification_token = ?", token).
		Where("email_verified = ?", true).
		Count(ctx)

	if err != nil {
		return false, fmt.Errorf("failed to check if token was used: %w", err)
	}

	return count > 0, nil
}

// MarkEmailAsVerified marks a user's email as verified and clears the verification token
func (r *Repository) MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error {
	result, err := r.db.NewUpdate().
		Model((*database.User)(nil)).
		Set("email_verified = ?", true).
		Set("email_verification_token = ?", nil).
		Set("email_verification_sent_at = ?", nil).
		Set("updated_at = NOW()").
		Where("id = ?", userID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to mark email as verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdatePassword updates a user's password hash
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result, err := r.db.NewUpdate().
		Model((*database.User)(nil)).
		Set("password_hash = ?", passwordHash).
		Set("updated_at = NOW()").
		Where("id = ?", userID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateVerificationToken regenerates verification token for resend
func (r *Repository) UpdateVerificationToken(ctx context.Context, userID uuid.UUID, token string) error {
	now := time.Now()
	result, err := r.db.NewUpdate().
		Model((*database.User)(nil)).
		Set("email_verification_token = ?", token).
		Set("email_verification_sent_at = ?", now).
		Set("updated_at = NOW()").
		Where("id = ?", userID).
		Where("email_verified = ?", false).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update verification token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// mapDBUserToModel converts database model to domain model
func mapDBUserToModel(dbu *database.User) *User {
	return &User{
		ID:                      dbu.ID,
		Email:                   dbu.Email,
		PasswordHash:            dbu.PasswordHash,
		EmailVerified:           dbu.EmailVerified,
		EmailVerificationToken:  dbu.EmailVerificationToken,
		EmailVerificationSentAt: dbu.EmailVerificationSentAt,
		CreatedAt:               dbu.CreatedAt,
		UpdatedAt:               dbu.UpdatedAt,
	}
}
