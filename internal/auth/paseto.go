package auth

import (
	"errors"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenClaims represents the claims stored in a PASETO token
type TokenClaims struct {
	UserID    string    `json:"user_id"` // UUID stored as string in token
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// PasetoService handles PASETO token creation and validation
// Uses v4.local (symmetric encryption with XChaCha20-Poly1305)
type PasetoService struct {
	symmetricKey paseto.V4SymmetricKey
}

func NewPasetoService(symmetricKey []byte) (*PasetoService, error) {
	if len(symmetricKey) != 32 {
		return nil, fmt.Errorf("symmetric key must be exactly 32 bytes, got %d", len(symmetricKey))
	}

	key, err := paseto.V4SymmetricKeyFromBytes(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create symmetric key: %w", err)
	}

	return &PasetoService{
		symmetricKey: key,
	}, nil
}

// CreateToken generates a new PASETO v4.local token with the given claims and duration
func (s *PasetoService) CreateToken(userID uuid.UUID, email string, duration time.Duration) (string, error) {
	now := time.Now()

	token := paseto.NewToken()
	token.SetIssuedAt(now)
	token.SetExpiration(now.Add(duration))
	token.SetString("user_id", userID.String())
	token.SetString("email", email)

	return token.V4Encrypt(s.symmetricKey, nil), nil
}

// VerifyToken validates a PASETO v4.local token and returns the claims
func (s *PasetoService) VerifyToken(tokenStr string) (*TokenClaims, error) {
	parser := paseto.NewParser()

	token, err := parser.ParseV4Local(s.symmetricKey, tokenStr, nil)
	if err != nil {
		// The parser checks expiration by default; distinguish expired from invalid
		if errors.Is(err, &paseto.RuleError{}) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	userID, err := token.GetString("user_id")
	if err != nil {
		return nil, ErrInvalidToken
	}

	email, err := token.GetString("email")
	if err != nil {
		return nil, ErrInvalidToken
	}

	issuedAt, err := token.GetIssuedAt()
	if err != nil {
		return nil, ErrInvalidToken
	}

	expiresAt, err := token.GetExpiration()
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &TokenClaims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}
