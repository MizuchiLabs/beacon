// Package api handles the API requests
package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v3"
	"github.com/mizuchilabs/beacon/internal/config"
)

type Server struct {
	mux *chi.Mux
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		mux: chi.NewMux(),
		cfg: cfg,
	}
}

func (s *Server) Start(ctx context.Context) error {
	defer s.cfg.Conn.Close()
	s.setupRoutes()

	// Start scheduler
	if err := s.cfg.Scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	logOpts := &httplog.Options{
		Level:           slog.LevelError,
		Schema:          httplog.SchemaOTEL,
		RecoverPanics:   true,
		LogRequestBody:  func(r *http.Request) bool { return s.cfg.Debug },
		LogResponseBody: func(r *http.Request) bool { return s.cfg.Debug },
	}

	// Create middleware chain
	chain := NewChain(
		s.WithCORS,
		httplog.RequestLogger(slog.Default(), logOpts),
	)

	server := &http.Server{
		Addr:              "0.0.0.0:" + s.cfg.ServerPort,
		Handler:           chain.Then(s.mux),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    8192, // 8KB
		TLSConfig: &tls.Config{
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: s.cfg.Insecure,
		},
	}

	// Channel to catch server errors
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		slog.Info("Server listening on", "address", "http://127.0.0.1:"+s.cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		slog.Info("Shutting down server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		s.cfg.Scheduler.Stop()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		return nil

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

func (s *Server) setupRoutes() {
	s.mux.Route("/api", func(r chi.Router) {
		// Monitor endpoints (read-only)
		r.Get("/monitors", s.ListMonitors)
		r.Get("/monitors/{id}", s.GetMonitor)

		// Status and stats endpoints
		r.Get("/monitors/stats", s.GetMonitorStats)

		// Health check endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
}
