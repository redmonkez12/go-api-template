package auth

import (
	"time"

	"github.com/google/uuid"
)

// TokenService defines the interface for token creation and validation.
// Implementations include PasetoService (PASETO v4.local) and JWTService (HS256).
type TokenService interface {
	CreateToken(userID uuid.UUID, email string, duration time.Duration) (string, error)
	VerifyToken(tokenStr string) (*TokenClaims, error)
}
