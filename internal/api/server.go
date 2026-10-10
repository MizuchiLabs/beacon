package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	"github.com/mizuchilabs/beacon/internal/scheduler"
	"github.com/mizuchilabs/beacon/web"
)

type Server struct {
	api huma.API
	mux *chi.Mux
}

func New(q *db.Queries, inc *incidents.Service, sched *scheduler.Service) (*Server, error) {
	mux := chi.NewRouter()

	mux.Use(httplog.RequestLogger(slog.Default(), &httplog.Options{
		RecoverPanics: true,
		Schema:        httplog.SchemaOTEL.Concise(logx.IsTerminal()),
		Skip: func(req *http.Request, respStatus int) bool {
			if respStatus >= http.StatusBadRequest {
				return false
			}
			p := req.URL.Path
			return p == "/healthz" || !strings.HasPrefix(p, "/api/")
		},
	}))
	mux.Use(cors.Default().Handler)
	mux.Use(middleware.RequestSize(1 << 20))
	mux.Use(securityHeaders())
	mux.Use(rateLimitAPI(100, time.Minute))
	mux.Use(middleware.CleanPath)

	api := humachi.New(mux, humaConfig())
	cfg, err := registerConfig(api)
	if err != nil {
		return nil, err
	}
	registerMonitors(api, q)
	registerIncidents(api, inc)
	registerNotify(api, q)
	registerHeartbeat(api, sched)
	registerBadges(api, q)

	mux.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Get("/incidents.atom", incidentFeed(inc, cfg.Title))
	mux.Handle("/*", web.Handler())
	return &Server{api: api, mux: mux}, nil
}

func humaConfig() huma.Config {
	cfg := huma.DefaultConfig("Beacon API", "1.0.0")
	cfg.CreateHooks = nil
	return cfg
}

// Spec builds the OpenAPI description of the API without starting a server.
func Spec() *huma.OpenAPI {
	server, _ := New(nil, nil, nil)
	return server.api.OpenAPI()
}

func (s *Server) Start(ctx context.Context, port string) error {
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    8192, // 8KB
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr, err)
	}
	slog.Info("server listening", "url", displayURL(ln.Addr()), "bind_address", ln.Addr().String())

	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		slog.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)

	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

func displayURL(a net.Addr) string {
	tcp, ok := a.(*net.TCPAddr)
	if !ok {
		return ""
	}
	host := tcp.IP.String()
	if tcp.IP.IsUnspecified() {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(tcp.Port))
}
