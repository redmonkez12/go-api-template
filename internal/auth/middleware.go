package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/redmonkez12/go-api-template/internal/httputil"

	"github.com/google/uuid"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	UserIDContextKey    ContextKey = "user_id"
	UserEmailContextKey ContextKey = "user_email"
)

// Middleware handles authentication for protected routes
type Middleware struct {
	tokenService TokenService
}

func NewMiddleware(tokenService TokenService) *Middleware {
	return &Middleware{tokenService: tokenService}
}

// RequireAuth is a middleware that validates the access token
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Priority 1: Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			} else {
				httputil.RespondErrorWithCode(w, "invalid authorization header format", httputil.CodeInvalidAuthHeader, http.StatusUnauthorized)
				return
			}
		}

		// Priority 2: Cookie (fallback)
		if token == "" {
			cookieToken, err := GetAccessTokenFromCookie(r)
			if err != nil {
				httputil.RespondErrorWithCode(w, "missing authentication", httputil.CodeMissingAuth, http.StatusUnauthorized)
				return
			}
			token = cookieToken
		}

		// Verify token
		claims, err := m.tokenService.VerifyToken(token)
		if err != nil {
			if err == ErrExpiredToken {
				httputil.RespondErrorWithCode(w, "token has expired", httputil.CodeTokenExpired, http.StatusUnauthorized)
				return
			}
			httputil.RespondErrorWithCode(w, "invalid token", httputil.CodeInvalidToken, http.StatusUnauthorized)
			return
		}

		// Parse UUID from claims
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			httputil.RespondErrorWithCode(w, "invalid user ID in token", httputil.CodeInvalidTokenUserID, http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		ctx = context.WithValue(ctx, UserEmailContextKey, claims.Email)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}

// GetUserEmailFromContext extracts the user email from the request context
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailContextKey).(string)
	return email, ok
}
