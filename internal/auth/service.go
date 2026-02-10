package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"go-api-template/internal/logging"
	"go-api-template/internal/user"
)

var (
	ErrInvalidCredentials       = errors.New("invalid email or password")
	ErrEmailRequired            = errors.New("email is required")
	ErrPasswordRequired         = errors.New("password is required")
	ErrPasswordTooShort         = errors.New("password must be at least 8 characters")
	ErrEmailNotVerified         = errors.New("email not verified, please check your inbox")
	ErrInvalidVerificationToken = errors.New("invalid verification token")
	ErrTokenExpired             = errors.New("verification token has expired")
	ErrEmailAlreadyVerified     = errors.New("email already verified")
	ErrInvalidEmailFormat       = errors.New("invalid email format")
)

// Argon2id parameters - tuned for security vs performance balance
// Time: 3, Memory: 64MB, Threads: 4, KeyLen: 32 bytes
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
)

// EmailService defines the interface for email operations
type EmailService interface {
	SendVerificationEmail(ctx context.Context, toEmail, token string) error
	SendPasswordResetEmail(ctx context.Context, toEmail, token string) error
}

// Service handles authentication business logic
type Service struct {
	userRepo             *user.Repository
	authRepo             RefreshTokenRepository
	passwordResetRepo    *PasswordResetRepository
	pasetoService        *PasetoService
	emailService         EmailService
	logger               *logging.Logger
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

func NewService(
	userRepo *user.Repository,
	authRepo RefreshTokenRepository,
	passwordResetRepo *PasswordResetRepository,
	pasetoService *PasetoService,
	emailService EmailService,
	logger *logging.Logger,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
) *Service {
	return &Service{
		userRepo:             userRepo,
		authRepo:             authRepo,
		passwordResetRepo:    passwordResetRepo,
		pasetoService:        pasetoService,
		emailService:         emailService,
		logger:               logger,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// Register creates a new user account and sends verification email
func (s *Service) Register(ctx context.Context, email, password string) (*user.User, error) {
	// Validate input
	if email == "" {
		return nil, ErrEmailRequired
	}
	if len(email) > 254 {
		return nil, ErrInvalidEmailFormat
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmailFormat
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Hash password using argon2id
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate verification token
	verificationToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Create user in database
	newUser, err := s.userRepo.Create(ctx, email, passwordHash, verificationToken)
	if err != nil {
		if errors.Is(err, user.ErrDuplicateEmail) {
			return nil, user.ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Send verification email in a goroutine (non-blocking)
	go func() {
		// Create a new context for the goroutine to avoid cancellation issues
		emailCtx := context.Background()
		if err := s.emailService.SendVerificationEmail(emailCtx, email, verificationToken); err != nil {
			// Log error but don't fail registration
			// User can request a new verification email later
			s.logger.Warn("failed to send verification email", "email", email, "error", err)
		}
	}()

	return newUser, nil
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, email, password string) (*AuthTokens, error) {
	// Validate input
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	// Get user from database
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if !s.verifyPassword(existingUser.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	// Check if email is verified
	if !existingUser.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, existingUser.ID, existingUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	// Get refresh token from database
	rt, err := s.authRepo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Validate refresh token
	if !rt.IsValid() {
		if rt.IsRevoked() {
			return nil, ErrRefreshTokenRevoked
		}
		if rt.IsExpired() {
			return nil, ErrRefreshTokenExpired
		}
	}

	// Revoke old refresh token before issuing new ones to prevent reuse
	if err := s.authRepo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Get user
	existingUser, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new tokens
	tokens, err := s.generateTokens(ctx, existingUser.ID, existingUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return s.authRepo.RevokeRefreshToken(ctx, refreshToken)
}

// VerifyEmail verifies a user's email using the verification token
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	// First, try to find user by token (only unverified users)
	existingUser, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Token not found in unverified users - check if it was already used
			alreadyVerified, checkErr := s.userRepo.CheckIfTokenAlreadyUsed(ctx, token)
			if checkErr == nil && alreadyVerified {
				return ErrEmailAlreadyVerified
			}
			// Token doesn't exist or is invalid
			return ErrInvalidVerificationToken
		}
		return fmt.Errorf("failed to find user by token: %w", err)
	}

	// Check if token has expired (24 hours)
	if existingUser.EmailVerificationSentAt == nil {
		return ErrTokenExpired
	}
	expirationTime := existingUser.EmailVerificationSentAt.Add(24 * time.Hour)
	if time.Now().After(expirationTime) {
		return ErrTokenExpired
	}

	// Mark email as verified
	if err := s.userRepo.MarkEmailAsVerified(ctx, existingUser.ID); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// generateTokens creates both access and refresh tokens
func (s *Service) generateTokens(ctx context.Context, userID uuid.UUID, email string) (*AuthTokens, error) {
	// Generate access token (short-lived)
	accessToken, err := s.pasetoService.CreateToken(userID, email, s.accessTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	// Generate refresh token (long-lived, random string)
	refreshToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(s.refreshTokenDuration)
	if err := s.authRepo.StoreRefreshToken(ctx, userID, refreshToken, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenDuration.Seconds()),
	}, nil
}

// hashPassword creates an argon2id hash of the password
func (s *Service) hashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash password with argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	// Encode as: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		encodedSalt,
		encodedHash,
	), nil
}

// verifyPassword checks if a password matches the stored hash
func (s *Service) verifyPassword(encodedHash, password string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	// Parse parameters
	var version int
	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Hash the input password with the same parameters
	inputHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(decodedHash)),
	)

	// Compare hashes using constant-time comparison
	return subtle.ConstantTimeCompare(decodedHash, inputHash) == 1
}

// generateRandomToken creates a cryptographically secure random token
func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RequestPasswordReset initiates the password reset process
// Always returns nil to prevent email enumeration attacks
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	// Get user by email
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		if errors.Is(err, user.ErrNotFound) {
			return nil
		}
		// Log error but return nil to prevent enumeration
		s.logger.Warn("failed to get user for password reset", "error", err)
		return nil
	}

	// Generate password reset token
	token, err := generateRandomToken()
	if err != nil {
		s.logger.Warn("failed to generate password reset token", "error", err)
		return nil
	}

	// Store token in Redis with 1-hour TTL
	if err := s.passwordResetRepo.StorePasswordResetToken(ctx, existingUser.ID, token); err != nil {
		s.logger.Warn("failed to store password reset token", "error", err)
		return nil
	}

	// Send password reset email in goroutine (non-blocking)
	go func() {
		emailCtx := context.Background()
		if err := s.emailService.SendPasswordResetEmail(emailCtx, email, token); err != nil {
			s.logger.Warn("failed to send password reset email", "email", email, "error", err)
		}
	}()

	return nil
}

// ResetPassword resets a user's password using a valid reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Validate password
	if newPassword == "" {
		return ErrPasswordRequired
	}
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Get user ID from token
	userID, err := s.passwordResetRepo.GetPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrPasswordResetTokenNotFound) {
			return ErrPasswordResetTokenNotFound
		}
		return fmt.Errorf("failed to get password reset token: %w", err)
	}

	// Hash new password
	passwordHash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password in database
	if err := s.userRepo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Delete used token
	if err := s.passwordResetRepo.DeletePasswordResetToken(ctx, token); err != nil {
		s.logger.Warn("failed to delete password reset token", "error", err)
	}

	// Revoke all refresh tokens for security
	if err := s.authRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		s.logger.Warn("failed to revoke all user tokens after password reset", "error", err)
	}

	return nil
}

// ResendVerificationEmail sends a new verification email to the user
// Always returns nil to prevent email enumeration attacks
func (s *Service) ResendVerificationEmail(ctx context.Context, email string) error {
	// Get user by email
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		if errors.Is(err, user.ErrNotFound) {
			return nil
		}
		// Log error but return nil to prevent enumeration
		s.logger.Warn("failed to get user for resend verification", "error", err)
		return nil
	}

	// Check if already verified
	if existingUser.EmailVerified {
		// Don't reveal that email is already verified
		return nil
	}

	// Generate new verification token
	token, err := generateRandomToken()
	if err != nil {
		s.logger.Warn("failed to generate verification token", "error", err)
		return nil
	}

	// Update verification token in database
	if err := s.userRepo.UpdateVerificationToken(ctx, existingUser.ID, token); err != nil {
		s.logger.Warn("failed to update verification token", "error", err)
		return nil
	}

	// Send verification email in goroutine (non-blocking)
	go func() {
		emailCtx := context.Background()
		if err := s.emailService.SendVerificationEmail(emailCtx, email, token); err != nil {
			s.logger.Warn("failed to resend verification email", "email", email, "error", err)
		}
	}()

	return nil
}
