package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrRefreshTokenNotFound        = errors.New("refresh token not found")
	ErrRefreshTokenRevoked         = errors.New("refresh token has been revoked")
	ErrRefreshTokenExpired         = errors.New("refresh token has expired")
	ErrPasswordResetTokenNotFound  = errors.New("password reset token not found or expired")
)

// hashToken creates a SHA-256 hash of the token for storage
// We store hashes instead of plain tokens for security
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
