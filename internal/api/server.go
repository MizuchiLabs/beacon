// Package api handles the API requests
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/rs/cors"

	"github.com/mizuchilabs/kata/logx"

	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
	"github.com/mizuchilabs/beacon/web"
)

type Server struct {
	api       huma.API
	mux       *chi.Mux
	q         *db.Queries
	incidents *incidents.Service
}

func New(
	q *db.Queries,
	inc *incidents.Service,
) (*Server, error) {
	mux := chi.NewRouter()

	if logx.IsTerminal() {
		mux.Use(middleware.Logger)
		mux.Use(middleware.Recoverer)
	} else {
		mux.Use(httplog.RequestLogger(slog.Default(), &httplog.Options{
			RecoverPanics: true,
			Schema:        httplog.SchemaOTEL,
			Skip: func(req *http.Request, respStatus int) bool {
				return respStatus < http.StatusBadRequest && req.URL.Path == "/healthz"
			},
		}))
	}
	mux.Use(cors.Default().Handler)
	mux.Use(middleware.RequestSize(1 << 20))
	mux.Use(securityHeaders())
	mux.Use(rateLimitAPI(100, time.Minute))
	mux.Use(middleware.CleanPath)
	return &Server{
		api:       newAPI(mux),
		mux:       mux,
		q:         q,
		incidents: inc,
	}, nil
}

func newAPI(mux *chi.Mux) huma.API {
	apiCfg := huma.DefaultConfig("Beacon API", "1.0.0")
	apiCfg.CreateHooks = nil
	return humachi.New(mux, apiCfg)
}

// Spec builds the OpenAPI description of the API without starting a server.
func Spec() *huma.OpenAPI {
	mux := chi.NewRouter()
	server := &Server{api: newAPI(mux), mux: mux}
	server.setupRoutes()
	return server.api.OpenAPI()
}

func (s *Server) Start(ctx context.Context, port string) error {
	s.setupRoutes()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           s.mux,
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

func (s *Server) setupRoutes() {
	// API routes
	NewConfigService(s.api)
	NewMonitorService(s.api, s.q)
	NewIncidentService(s.api, s.incidents)
	NewNotifyService(s.api, s.q)

	// Liveness: Process is alive and not deadlocked
	s.mux.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.mux.Handle("/*", web.Handler())
}
