// Package api handles the API requests
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v3"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/web"
	"github.com/vearutop/statigz"
	"github.com/vearutop/statigz/brotli"
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
	s.setupRoutes()

	// Start scheduler
	if err := s.cfg.Scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Start incident syncer if enabled
	if err := s.cfg.Incidents.Start(ctx); err != nil {
		return fmt.Errorf("failed to start incident syncer: %w", err)
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
		// read-only endpoints
		r.Get("/monitors", s.GetMonitors)
		r.Get("/config", s.GetConfig)
		r.Get("/incidents", s.GetIncidents)
		r.Get("/incidents/{id}", s.GetIncident)

		// Push notification endpoints
		r.Post("/monitor/{id}/subscribe", s.SubscribeToPushNotifications)
		r.Post("/monitor/{id}/unsubscribe", s.UnsubscribeFromPushNotifications)
		r.Get("/vapid-public-key", s.GetVAPIDPublicKey)

		// Health check endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// Static files
	s.mux.Handle("/*", statigz.FileServer(
		web.StaticFS,
		brotli.AddEncoding,
		statigz.FSPrefix("build"),
	))
}
