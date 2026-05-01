// Package api handles the API requests
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/web"
	"github.com/vearutop/statigz"
)

type Server struct {
	mux *http.ServeMux
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		mux: http.NewServeMux(),
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

	chain := NewChain(
		s.WithCORS,
		s.WithLogger,
		WithRateLimit,
		WithBodyLimit,
		WithSecurityHeaders,
	)
	server := &http.Server{
		Addr:              ":" + s.cfg.ServerPort,
		Handler:           chain.Then(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    8192, // 8KB
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Server listening on", "port", s.cfg.ServerPort)
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
	s.mux.HandleFunc("GET /api/monitors", s.GetMonitors)
	s.mux.HandleFunc("GET /api/config", s.GetConfig)
	s.mux.HandleFunc("GET /api/incidents", s.GetIncidents)
	s.mux.HandleFunc("GET /api/incidents/{id}", s.GetIncident)

	// Push notifications
	s.mux.HandleFunc("POST /api/monitor/{id}/subscribe", s.SubscribeToPushNotifications)
	s.mux.HandleFunc("POST /api/monitor/{id}/unsubscribe", s.UnsubscribeFromPushNotifications)
	s.mux.HandleFunc("GET /api/vapid-public-key", s.GetVAPIDPublicKey)

	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Static files
	s.mux.Handle("/", statigz.FileServer(web.StaticFS, statigz.FSPrefix("build")))

	if s.cfg.Debug {
		s.mux.HandleFunc("/debug/pprof/", pprof.Index)
		s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}
