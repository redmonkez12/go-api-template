package logging

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional context
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger
// In production, this could be configured to use different outputs/formats
func NewLogger(isDevelopment bool) *Logger {
	var handler slog.Handler

	if isDevelopment {
		// JSON format for easy parsing by Loki
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		// Production: still JSON but maybe different log level
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithFields adds fields to the logger context
func (l *Logger) WithFields(fields map[string]any) *Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	return &Logger{
		Logger: l.Logger.With(args...),
	}
}
