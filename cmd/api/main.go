package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	_ "github.com/redmonkez12/go-api-template/docs" // Swagger docs (generated)
	"github.com/redmonkez12/go-api-template/internal/auth"
	"github.com/redmonkez12/go-api-template/internal/config"
	"github.com/redmonkez12/go-api-template/internal/email"
	httpServer "github.com/redmonkez12/go-api-template/internal/http"
	"github.com/redmonkez12/go-api-template/internal/logging"
	"github.com/redmonkez12/go-api-template/internal/ratelimit"
	"github.com/redmonkez12/go-api-template/internal/user"
)

// @title           Go API Template
// @version         1.0
// @description     A production-ready Go REST API template with authentication, email verification, and observability.

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the access token.

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger := logging.NewLogger(cfg.Server.IsDevelopment())
	logger.Info("starting application",
		"env", cfg.Server.Env,
		"port", cfg.Server.Port,
	)

	// Initialize database connection
	db, err := initDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := initRedis(cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	userRepo := user.NewRepository(db)
	authRepo := auth.NewRedisRepository(redisClient)
	passwordResetRepo := auth.NewPasswordResetRepository(redisClient)

	// Initialize rate limiter
	rateLimiter := ratelimit.NewLimiter(redisClient)

	// Initialize PASETO service
	pasetoService, err := auth.NewPasetoService(cfg.Auth.PasetoKey)
	if err != nil {
		return fmt.Errorf("failed to initialize PASETO service: %w", err)
	}

	// Initialize email service
	emailService := email.NewService(
		cfg.Email.SMTPHost,
		cfg.Email.SMTPPort,
		cfg.Email.SMTPUser,
		cfg.Email.SMTPPassword,
		cfg.Email.FrontendURL,
	)

	// Initialize auth service
	authService := auth.NewService(
		userRepo,
		authRepo,
		passwordResetRepo,
		pasetoService,
		emailService,
		logger,
		cfg.Auth.AccessTokenDuration,
		cfg.Auth.RefreshTokenDuration,
	)

	// Initialize HTTP handlers
	authHandler := auth.NewHandler(
		authService,
		rateLimiter,
		logger,
		!cfg.Server.IsDevelopment(), // isProduction
		cfg.Auth.AccessTokenDuration,
		cfg.Auth.RefreshTokenDuration,
	)
	authMiddleware := auth.NewMiddleware(pasetoService)

	// Initialize router
	router := httpServer.NewRouter(cfg, authHandler, authMiddleware, logger)

	// Initialize HTTP server
	serverAddr := ":" + cfg.Server.Port
	server := httpServer.NewServer(
		serverAddr,
		router,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
	)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Printf("Received signal: %v", sig)

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}

// initDB initializes the database connection and returns a Bun DB instance
func initDB(cfg config.DatabaseConfig) (*bun.DB, error) {
	sqlDB, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	// Create Bun DB wrapper
	db := bun.NewDB(sqlDB, pgdialect.New())

	return db, nil
}

// initRedis initializes the Redis connection and returns a Redis client
func initRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Verify connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}
