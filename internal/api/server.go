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
	"github.com/vearutop/statigz"

	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
	"github.com/mizuchilabs/beacon/web"
)

type Server struct {
	api       huma.API
	mux       *http.ServeMux
	cfg       *Config
	q         *db.Queries
	incidents *incidents.IncidentManager
	debug     bool
	rootCtx   context.Context
}

// Spec builds the OpenAPI description of the API without starting a server.
func Spec() *huma.OpenAPI {
	mux := http.NewServeMux()
	server := &Server{api: newAPI(mux), mux: mux, cfg: &Config{}}
	server.setupRoutes()
	return server.api.OpenAPI()
}

func NewServer(
	ctx context.Context,
	q *db.Queries,
	inc *incidents.IncidentManager,
) (*Server, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	return &Server{
		api:       newAPI(mux),
		mux:       mux,
		cfg:       cfg,
		q:         q,
		incidents: inc,
		// logx owns the log level, an enabled debug level is the switch
		debug:   slog.Default().Enabled(ctx, slog.LevelDebug),
		rootCtx: ctx,
	}, nil
}

func newAPI(mux *http.ServeMux) huma.API {
	apiCfg := huma.DefaultConfig("Beacon API", "1.0.0")
	apiCfg.CreateHooks = nil
	return humago.New(mux, apiCfg)
}

func (s *Server) Start() error {
	s.setupRoutes()

	chain := NewChain(
		s.WithCORS,
		s.WithLogger,
		WithRateLimit,
		WithBodyLimit,
		WithSecurityHeaders,
	)
	server := &http.Server{
		Addr:              ":" + s.cfg.Port,
		Handler:           chain.Then(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    8192, // 8KB
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Server listening on", "port", s.cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-s.rootCtx.Done():
		slog.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

func (s *Server) setupRoutes() {
	// API routes, each service registers its own operations on the huma API
	NewConfigService(s.api, s.cfg, s.incidents)
	NewMonitorService(s.api, s.cfg, s.q)
	NewIncidentService(s.api, s.incidents)
	NewNotifyService(s.api, s.q)

	// Plain mux routes outside the OpenAPI spec
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Static files
	s.mux.Handle("/", statigz.FileServer(web.StaticFS, statigz.FSPrefix("build")))

	if s.debug {
		s.mux.HandleFunc("/debug/pprof/", pprof.Index)
		s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}
