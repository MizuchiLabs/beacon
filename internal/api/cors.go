package api

import (
	"net/http"
	"time"

	"github.com/rs/cors"
)

func (s *Server) WithCORS(h http.Handler) http.Handler {
	allowedOrigins := []string{
		"http://127.0.0.1:" + s.cfg.ServerPort,
		"http://localhost:" + s.cfg.ServerPort,
		"http://localhost:5173",
	}

	return cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           int(2 * time.Hour / time.Second),
	}).Handler(h)
}
