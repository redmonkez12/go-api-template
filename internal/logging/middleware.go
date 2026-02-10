package logging

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// LoggerContextKey is the key for the logger in the request context
	LoggerContextKey ContextKey = "logger"
)

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status
		written:        false,
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestLogger is a middleware that logs HTTP requests
func RequestLogger(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get or generate request ID (chi middleware should have already set this)
			requestID := middleware.GetReqID(r.Context())

			// Create a logger with request context
			reqLogger := logger.WithFields(map[string]any{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  r.RemoteAddr,
			})

			// Log request start
			reqLogger.Info("request started")

			// Add logger to request context for use in handlers
			ctx := context.WithValue(r.Context(), LoggerContextKey, reqLogger)

			// Wrap response writer to capture status code
			wrapped := newResponseWriter(w)

			// Process request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Calculate duration
			duration := time.Since(start)

			// Log request completion with appropriate level
			logLevel := slog.LevelInfo
			if wrapped.statusCode >= 500 {
				logLevel = slog.LevelError
			} else if wrapped.statusCode >= 400 {
				logLevel = slog.LevelWarn
			}

			reqLogger.Log(r.Context(), logLevel, "request completed",
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

// GetLoggerFromContext retrieves the logger from the request context
func GetLoggerFromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*Logger); ok {
		return logger
	}
	// Fallback to a default logger if not found
	return NewLogger(true)
}
