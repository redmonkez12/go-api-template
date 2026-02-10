package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port            string
	Env             string // dev or prod
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	TrustedOrigins  []string // CORS allowed origins for cookie auth
}

type DatabaseConfig struct {
	Host           string
	Port           string
	User           string
	Password       string
	DBName         string
	SSLMode        string
	ChannelBinding string // "require" for Neon DB, empty for local
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AuthConfig struct {
	// PASETO symmetric key (must be 32 bytes for v4.local)
	PasetoKey            []byte
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FrontendURL  string // Frontend URL for verification links
}

// Load reads configuration from environment variables
// Call godotenv.Load() before this if using .env file
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Env:             getEnv("APP_ENV", "dev"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
			TrustedOrigins:  getSliceEnv("TRUSTED_ORIGINS", []string{"http://localhost:3000"}),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           getEnv("DB_PORT", "5432"),
			User:           getEnv("DB_USER", "postgres"),
			Password:       getEnv("DB_PASSWORD", "postgres"),
			DBName:         getEnv("DB_NAME", "goapi"),
			SSLMode:        getEnv("DB_SSLMODE", "disable"),
			ChannelBinding: getEnv("DB_CHANNEL_BINDING", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			PasetoKey:            []byte(getEnv("PASETO_KEY", "")),
			AccessTokenDuration:  getDurationEnv("ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: getDurationEnv("REFRESH_TOKEN_DURATION", 7*24*time.Hour),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASS", ""),
			FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
	}

	// Validate PASETO key length (must be 32 bytes for v4.local)
	if len(cfg.Auth.PasetoKey) != 32 {
		return nil, fmt.Errorf("PASETO_KEY must be exactly 32 bytes, got %d", len(cfg.Auth.PasetoKey))
	}

	return cfg, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)

	// Add channel_binding if configured (required for Neon DB)
	if c.ChannelBinding != "" {
		connStr += fmt.Sprintf(" channel_binding=%s", c.ChannelBinding)
	}

	return connStr
}

// Address returns Redis connection address (host:port)
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// IsDevelopment returns true if the environment is set to dev
func (c *ServerConfig) IsDevelopment() bool {
	return c.Env == "dev"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return time.Duration(seconds) * time.Second
}

func getSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Split by comma and trim whitespace
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}
