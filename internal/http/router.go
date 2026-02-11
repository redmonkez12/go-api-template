package http

import (
	"log"
	"net/http"

	"github.com/redmonkez12/go-api-template/internal/auth"
	"github.com/redmonkez12/go-api-template/internal/config"
	"github.com/redmonkez12/go-api-template/internal/httputil"
	"github.com/redmonkez12/go-api-template/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter creates and configures the HTTP router
func NewRouter(cfg *config.Config, authHandler *auth.Handler, authMiddleware *auth.Middleware, logger *logging.Logger) *chi.Mux {
	r := chi.NewRouter()

	// CORS - must be first
	if len(cfg.Server.TrustedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.Server.TrustedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           300, // 5 minutes
		}))
	}

	// Global middleware
	r.Use(SecurityHeaders)               // Security headers on all responses
	r.Use(middleware.Recoverer)          // Recover from panics
	r.Use(middleware.RequestID)          // Add request ID
	r.Use(middleware.RealIP)             // Set RemoteAddr to real IP
	r.Use(logging.RequestLogger(logger)) // Structured logging with request context
	r.Use(middleware.Compress(5))        // Compress responses

	// Public routes
	r.Get("/health", handleHealth)

	// Swagger UI - only in development
	// Production builds will not have this route at all
	if cfg.Server.IsDevelopment() {
		log.Println("Swagger UI enabled at /swagger/*")
		r.Get("/swagger/*", httpSwagger.WrapHandler)
	} else {
		log.Println("Swagger UI disabled (production mode)")
	}

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
		r.Get("/verify-email", authHandler.VerifyEmail)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
		r.Post("/resend-verification", authHandler.ResendVerificationEmail)
	})

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
	})

	return r
}

// handleHealth is a simple health check endpoint
// @Summary      Health check
// @Description  Check if the API is running
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health [get]
func handleHealth(w http.ResponseWriter, r *http.Request) {
	httputil.RespondJSON(w, map[string]string{"status": "api is running"}, http.StatusOK)
}

