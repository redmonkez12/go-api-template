package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server wraps the HTTP server with graceful shutdown
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new HTTP server
func NewServer(addr string, handler http.Handler, readTimeout, writeTimeout time.Duration) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

// Start begins listening and serving HTTP requests
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
