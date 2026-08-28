// Package api handles the API requests
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/web"
	"github.com/vearutop/statigz"
)

type Server struct {
	mux *http.ServeMux
	api huma.API
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	mux := http.NewServeMux()
	apiCfg := huma.DefaultConfig("Beacon API", "1.0.0")
	apiCfg.CreateHooks = nil
	api := humago.New(mux, apiCfg)
	s := &Server{mux: mux, api: api, cfg: cfg}
	s.setupRoutes()
	return s
}

func (s *Server) OpenAPI() *huma.OpenAPI {
	return s.api.OpenAPI()
}

func (s *Server) setupRoutes() {
	// API routes, each service registers its own operations on the huma API
	NewConfigService(s.api, s.cfg)
	NewMonitorService(s.api, s.cfg)
	NewIncidentService(s.api, s.cfg)
	NewNotifyService(s.api, s.cfg)

	// Plain mux routes outside the OpenAPI spec
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) Start(ctx context.Context) error {
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
		slog.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}
