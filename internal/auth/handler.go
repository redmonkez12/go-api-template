package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redmonkez12/go-api-template/internal/httputil"
	"github.com/redmonkez12/go-api-template/internal/logging"
	"github.com/redmonkez12/go-api-template/internal/ratelimit"
	"github.com/redmonkez12/go-api-template/internal/user"
)

// Handler contains HTTP handlers for authentication endpoints
type Handler struct {
	service          *Service
	rateLimiter      *ratelimit.Limiter
	logger           *logging.Logger
	isProduction     bool
	accessDuration   time.Duration
	refreshDuration  time.Duration
}

func NewHandler(service *Service, rateLimiter *ratelimit.Limiter, logger *logging.Logger, isProduction bool, accessDuration, refreshDuration time.Duration) *Handler {
	return &Handler{
		service:          service,
		rateLimiter:      rateLimiter,
		logger:           logger,
		isProduction:     isProduction,
		accessDuration:   accessDuration,
		refreshDuration:  refreshDuration,
	}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest represents the token refresh request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	User    UserResponse `json:"user"`
	Message string       `json:"message"`
}

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// ForgotPasswordRequest represents the password reset request
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents the password reset confirmation
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResendVerificationRequest represents the resend verification email request
type ResendVerificationRequest struct {
	Email string `json:"email"`
}

// Register handles user registration
// @Summary      Register a new user
// @Description  Create a new user account with email and password. A verification email will be sent.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration credentials"
// @Success      201 {object} RegisterResponse
// @Failure      400 {object} ErrorResponse "Invalid request or validation error"
// @Failure      409 {object} ErrorResponse "Email already exists"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	// Rate limit by IP
	ip := getClientIP(r)
	exceeded, err := h.rateLimiter.CheckIPRateLimitWithPurpose(r.Context(), ip, "register")
	if err != nil {
		logger.Error("failed to check IP rate limit", "error", err.Error())
	} else if exceeded {
		logger.Warn("IP rate limit exceeded for register", "ip", ip)
		respondError(w, "too many requests, please try again later", httputil.CodeTooManyRequests, http.StatusTooManyRequests)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid registration request body", "error", err.Error())
		respondError(w, "invalid request body", httputil.CodeInvalidRequestBody, http.StatusBadRequest)
		return
	}

	logger = logger.WithFields(map[string]any{"email": req.Email})

	// Record IP request for rate limiting
	if err := h.rateLimiter.RecordIPRequestWithPurpose(r.Context(), ip, "register"); err != nil {
		logger.Error("failed to record IP request", "error", err.Error())
	}

	// Register user
	newUser, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrDuplicateEmail) {
			logger.Warn("registration failed: email already exists")
			respondError(w, "email already exists", httputil.CodeEmailAlreadyExists, http.StatusConflict)
			return
		}
		if errors.Is(err, ErrEmailRequired) {
			logger.Warn("registration failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodeEmailRequired, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrPasswordRequired) {
			logger.Warn("registration failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodePasswordRequired, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrPasswordTooShort) {
			logger.Warn("registration failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodePasswordTooShort, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrInvalidEmailFormat) {
			logger.Warn("registration failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodeInvalidEmailFormat, http.StatusBadRequest)
			return
		}
		logger.Error("registration failed: internal error", "error", err.Error())
		respondError(w, "failed to register user", httputil.CodeInternalError, http.StatusInternalServerError)
		return
	}

	logger.Info("user registered successfully", "user_id", newUser.ID)

	userResponse := UserResponse{
		ID:    newUser.ID,
		Email: newUser.Email,
	}

	respondJSON(w, RegisterResponse{
		User:    userResponse,
		Message: "Registration successful. Please check your email to verify your account.",
	}, http.StatusCreated)
}

// Login handles user login
// @Summary      User login
// @Description  Authenticate user and receive access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} AuthTokens
// @Failure      400 {object} ErrorResponse "Invalid request body"
// @Failure      401 {object} ErrorResponse "Invalid credentials"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	// Rate limit by IP
	ip := getClientIP(r)
	exceeded, err := h.rateLimiter.CheckIPRateLimitWithPurpose(r.Context(), ip, "login")
	if err != nil {
		logger.Error("failed to check IP rate limit", "error", err.Error())
	} else if exceeded {
		logger.Warn("IP rate limit exceeded for login", "ip", ip)
		respondError(w, "too many requests, please try again later", httputil.CodeTooManyRequests, http.StatusTooManyRequests)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid login request body", "error", err.Error())
		respondError(w, "invalid request body", httputil.CodeInvalidRequestBody, http.StatusBadRequest)
		return
	}

	logger = logger.WithFields(map[string]any{"email": req.Email})

	// Record IP request for rate limiting
	if err := h.rateLimiter.RecordIPRequestWithPurpose(r.Context(), ip, "login"); err != nil {
		logger.Error("failed to record IP request", "error", err.Error())
	}

	tokens, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			logger.Warn("login failed: invalid credentials")
			respondError(w, "invalid email or password", httputil.CodeInvalidCredentials, http.StatusUnauthorized)
			return
		}
		if errors.Is(err, ErrEmailNotVerified) {
			logger.Warn("login failed: email not verified")
			respondError(w, "email not verified, please check your inbox", httputil.CodeEmailNotVerified, http.StatusForbidden)
			return
		}
		logger.Error("login failed: internal error", "error", err.Error())
		respondError(w, "failed to login", httputil.CodeInternalError, http.StatusInternalServerError)
		return
	}

	logger.Info("user logged in successfully")

	// Set cookies if request is from browser
	if ShouldUseCookies(r) {
		SetAuthCookies(w, tokens.AccessToken, tokens.RefreshToken, h.isProduction, h.accessDuration, h.refreshDuration)
		// Don't return tokens in response body when using cookies
		respondJSON(w, map[string]string{
			"message": "logged in successfully",
		}, http.StatusOK)
	} else {
		// Return tokens in response body for non-browser clients
		respondJSON(w, tokens, http.StatusOK)
	}
}

// Refresh handles access token refresh
// @Summary      Refresh access token
// @Description  Use a refresh token to get a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200 {object} AuthTokens
// @Failure      400 {object} ErrorResponse "Invalid request body"
// @Failure      401 {object} ErrorResponse "Invalid or expired refresh token"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	// Try to get refresh token from JSON body first
	var refreshToken string
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
		refreshToken = req.RefreshToken
	}

	// Fallback to cookie if body empty/invalid
	if refreshToken == "" {
		cookieToken, err := GetRefreshTokenFromCookie(r)
		if err == nil {
			refreshToken = cookieToken
		}
	}

	if refreshToken == "" {
		logger.Warn("refresh token missing from both body and cookie")
		respondError(w, "refresh token required", httputil.CodeRefreshTokenRequired, http.StatusBadRequest)
		return
	}

	// Trim whitespace that might have been accidentally added
	refreshToken = strings.TrimSpace(refreshToken)

	tokens, err := h.service.RefreshAccessToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrRefreshTokenRevoked) || errors.Is(err, ErrRefreshTokenExpired) {
			logger.Warn("token refresh failed: invalid or expired token", "error", err.Error())
			respondError(w, "invalid or expired refresh token", httputil.CodeInvalidRefreshToken, http.StatusUnauthorized)
			return
		}
		logger.Error("token refresh failed: internal error", "error", err.Error())
		respondError(w, "failed to refresh token", httputil.CodeInternalError, http.StatusInternalServerError)
		return
	}

	logger.Info("access token refreshed successfully")

	// Set cookies if request is from browser
	if ShouldUseCookies(r) {
		SetAuthCookies(w, tokens.AccessToken, tokens.RefreshToken, h.isProduction, h.accessDuration, h.refreshDuration)
		// Don't return tokens in response body when using cookies
		respondJSON(w, map[string]string{
			"message": "token refreshed successfully",
		}, http.StatusOK)
	} else {
		// Return tokens in response body for non-browser clients
		respondJSON(w, tokens, http.StatusOK)
	}
}

// VerifyEmail handles email verification
// @Summary      Verify email address
// @Description  Verify a user's email address using the verification token sent via email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        token query string true "Verification token"
// @Success      200 {object} map[string]string
// @Failure      400 {object} ErrorResponse "Invalid, expired, or already used token"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/verify-email [get]
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		logger.Warn("email verification failed: token missing")
		respondError(w, "verification token required", httputil.CodeVerificationTokenRequired, http.StatusBadRequest)
		return
	}

	err := h.service.VerifyEmail(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			logger.Warn("email verification failed: token expired")
			httputil.RespondErrorWithCode(w, "Verification link has expired. Please request a new one.", httputil.CodeTokenExpired, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrEmailAlreadyVerified) {
			logger.Warn("email verification failed: already verified")
			httputil.RespondErrorWithCode(w, "This email is already verified. You can login now.", httputil.CodeAlreadyVerified, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrInvalidVerificationToken) {
			logger.Warn("email verification failed: invalid token")
			httputil.RespondErrorWithCode(w, "Invalid verification token.", httputil.CodeVerificationFailed, http.StatusBadRequest)
			return
		}
		logger.Error("email verification failed: internal error", "error", err.Error())
		respondError(w, "failed to verify email", httputil.CodeInternalError, http.StatusInternalServerError)
		return
	}

	logger.Info("email verified successfully")

	respondJSON(w, map[string]string{
		"message": "Email verified successfully. You can now login.",
	}, http.StatusOK)
}

// Logout handles user logout
// @Summary      User logout
// @Description  Logout user by revoking refresh token and clearing cookies
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest false "Optional refresh token"
// @Success      200 {object} map[string]string
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	// Get refresh token from either source
	var refreshToken string
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
		refreshToken = req.RefreshToken
	}
	if refreshToken == "" {
		cookieToken, _ := GetRefreshTokenFromCookie(r)
		refreshToken = cookieToken
	}

	// Revoke refresh token if provided
	if refreshToken != "" {
		if err := h.service.RevokeRefreshToken(r.Context(), refreshToken); err != nil {
			logger.Warn("failed to revoke refresh token", "error", err)
			// Continue - still clear cookies
		}
	}

	// Clear cookies
	ClearAuthCookies(w)

	logger.Info("user logged out successfully")

	respondJSON(w, map[string]string{"message": "logged out"}, http.StatusOK)
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, data any, statusCode int) {
	httputil.RespondJSON(w, data, statusCode)
}

// respondError sends an error response with a machine-readable code
func respondError(w http.ResponseWriter, message string, code string, statusCode int) {
	httputil.RespondErrorWithCode(w, message, code, statusCode)
}

// ForgotPassword handles password reset requests
// @Summary      Request password reset
// @Description  Send a password reset link to the user's email. Always returns success to prevent email enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ForgotPasswordRequest true "Email address"
// @Success      200 {object} map[string]string
// @Failure      400 {object} ErrorResponse "Invalid request body"
// @Failure      429 {object} ErrorResponse "Too many requests"
// @Router       /auth/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid forgot password request body", "error", err.Error())
		respondError(w, "invalid request body", httputil.CodeInvalidRequestBody, http.StatusBadRequest)
		return
	}

	// Get client IP for rate limiting
	ip := getClientIP(r)

	// Check IP rate limit (10 req/15 min)
	exceeded, err := h.rateLimiter.CheckIPRateLimit(r.Context(), ip)
	if err != nil {
		logger.Error("failed to check IP rate limit", "error", err.Error())
		// Continue despite error to avoid blocking legitimate requests
	} else if exceeded {
		logger.Warn("IP rate limit exceeded", "ip", ip)
		respondError(w, "too many requests, please try again later", httputil.CodeTooManyRequests, http.StatusTooManyRequests)
		return
	}

	// Check email cooldown (2 min)
	onCooldown, err := h.rateLimiter.CheckEmailCooldown(r.Context(), req.Email)
	if err != nil {
		logger.Error("failed to check email cooldown", "error", err.Error())
		// Continue despite error
	} else if onCooldown {
		logger.Warn("email on cooldown", "email", req.Email)
		respondError(w, "please wait before requesting another reset", httputil.CodeCooldownActive, http.StatusTooManyRequests)
		return
	}

	// Record IP request for rate limiting
	if err := h.rateLimiter.RecordIPRequest(r.Context(), ip); err != nil {
		logger.Error("failed to record IP request", "error", err.Error())
	}

	// Set email cooldown
	if err := h.rateLimiter.SetEmailCooldown(r.Context(), req.Email); err != nil {
		logger.Error("failed to set email cooldown", "error", err.Error())
	}

	// Process request (always returns nil for security)
	_ = h.service.RequestPasswordReset(r.Context(), req.Email)

	// Always return success (prevent email enumeration)
	respondJSON(w, map[string]string{
		"message": "If an account exists with that email, a password reset link has been sent.",
	}, http.StatusOK)
}

// ResetPassword handles password reset with token
// @Summary      Reset password
// @Description  Reset a user's password using a valid reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "Reset token and new password"
// @Success      200 {object} map[string]string
// @Failure      400 {object} ErrorResponse "Invalid request or token"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid reset password request body", "error", err.Error())
		respondError(w, "invalid request body", httputil.CodeInvalidRequestBody, http.StatusBadRequest)
		return
	}

	err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrPasswordResetTokenNotFound) {
			logger.Warn("password reset failed: invalid or expired token")
			respondError(w, "invalid or expired reset token", httputil.CodeInvalidResetToken, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrPasswordRequired) {
			logger.Warn("password reset failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodePasswordRequired, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrPasswordTooShort) {
			logger.Warn("password reset failed: validation error", "error", err.Error())
			respondError(w, err.Error(), httputil.CodePasswordTooShort, http.StatusBadRequest)
			return
		}
		logger.Error("password reset failed: internal error", "error", err.Error())
		respondError(w, "failed to reset password", httputil.CodeInternalError, http.StatusInternalServerError)
		return
	}

	logger.Info("password reset successfully")

	respondJSON(w, map[string]string{
		"message": "Password reset successfully. You can now login with your new password.",
	}, http.StatusOK)
}

// ResendVerificationEmail handles resending verification email
// @Summary      Resend verification email
// @Description  Send a new verification email to the user. Always returns success to prevent email enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResendVerificationRequest true "Email address"
// @Success      200 {object} map[string]string
// @Failure      400 {object} ErrorResponse "Invalid request body"
// @Failure      429 {object} ErrorResponse "Too many requests"
// @Router       /auth/resend-verification [post]
func (h *Handler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLoggerFromContext(r.Context())

	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid resend verification request body", "error", err.Error())
		respondError(w, "invalid request body", httputil.CodeInvalidRequestBody, http.StatusBadRequest)
		return
	}

	// Get client IP for rate limiting
	ip := getClientIP(r)

	// Check IP rate limit (10 req/15 min)
	exceeded, err := h.rateLimiter.CheckIPRateLimit(r.Context(), ip)
	if err != nil {
		logger.Error("failed to check IP rate limit", "error", err.Error())
		// Continue despite error
	} else if exceeded {
		logger.Warn("IP rate limit exceeded", "ip", ip)
		respondError(w, "too many requests, please try again later", httputil.CodeTooManyRequests, http.StatusTooManyRequests)
		return
	}

	// Check email cooldown (2 min)
	onCooldown, err := h.rateLimiter.CheckEmailCooldown(r.Context(), req.Email)
	if err != nil {
		logger.Error("failed to check email cooldown", "error", err.Error())
		// Continue despite error
	} else if onCooldown {
		logger.Warn("email on cooldown", "email", req.Email)
		respondError(w, "please wait before requesting another email", httputil.CodeCooldownActive, http.StatusTooManyRequests)
		return
	}

	// Record IP request for rate limiting
	if err := h.rateLimiter.RecordIPRequest(r.Context(), ip); err != nil {
		logger.Error("failed to record IP request", "error", err.Error())
	}

	// Set email cooldown
	if err := h.rateLimiter.SetEmailCooldown(r.Context(), req.Email); err != nil {
		logger.Error("failed to set email cooldown", "error", err.Error())
	}

	// Process request (always returns nil for security)
	_ = h.service.ResendVerificationEmail(r.Context(), req.Email)

	// Always return success (prevent email enumeration)
	respondJSON(w, map[string]string{
		"message": "If your email is registered and not verified, a new verification link has been sent.",
	}, http.StatusOK)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (behind proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	// RemoteAddr format is "IP:port", extract just the IP
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
