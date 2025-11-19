package api

import (
	"net/http"
	"time"

	"github.com/rs/cors"
)

func (s *Server) WithCORS(h http.Handler) http.Handler {
	// allowedOrigins := []string{
	// 	util.OriginOnly(s.cfg.BaseURL),
	// 	util.OriginOnly(s.cfg.FrontendURL),
	// }

	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           int(2 * time.Hour / time.Second),
	}).Handler(h)
}
