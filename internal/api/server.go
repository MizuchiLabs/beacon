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
	q         *db.Queries
	incidents *incidents.Service
	debug     bool
}

func New(
	ctx context.Context,
	q *db.Queries,
	inc *incidents.Service,
) (*Server, error) {
	mux := http.NewServeMux()
	return &Server{
		api:       newAPI(mux),
		mux:       mux,
		q:         q,
		incidents: inc,
		debug:     slog.Default().Enabled(ctx, slog.LevelDebug),
	}, nil
}

func newAPI(mux *http.ServeMux) huma.API {
	apiCfg := huma.DefaultConfig("Beacon API", "1.0.0")
	apiCfg.CreateHooks = nil
	return humago.New(mux, apiCfg)
}

// Spec builds the OpenAPI description of the API without starting a server.
func Spec() *huma.OpenAPI {
	mux := http.NewServeMux()
	server := &Server{api: newAPI(mux), mux: mux}
	server.setupRoutes()
	return server.api.OpenAPI()
}

func (s *Server) Start(ctx context.Context, port string) error {
	s.setupRoutes()

	chain := NewChain(
		WithCORS(port),
		s.WithLogger,
		WithRateLimit,
		WithBodyLimit,
		WithSecurityHeaders,
	)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           chain.Then(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    8192, // 8KB
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Server listening on", "port", port)
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

func (s *Server) setupRoutes() {
	// API routes, each service registers its own operations on the huma API
	NewConfigService(s.api)
	NewMonitorService(s.api, s.q)
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
