package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/cors"
	"golang.org/x/time/rate"
)

const (
	// RPS is the general per-IP rate limit.
	RPS = 30
	// Burst is the max burst size on top of RPS.
	Burst = 50

	// MaxBodySize is the request body size limit.
	MaxBodySize = 1 << 20
)

func (s *Server) WithCORS(h http.Handler) http.Handler {
	allowedOrigins := []string{
		"http://127.0.0.1:" + s.cfg.Port,
		"http://localhost:" + s.cfg.Port,
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

// WithBodyLimit restricts the size of incoming request bodies.
func WithBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > MaxBodySize {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)
		next.ServeHTTP(w, r)
	})
}

// WithSecurityHeaders adds basic secure HTTP headers.
func WithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().
			Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' data:")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// forwardedFor returns the client address from X-Forwarded-For when the direct
// peer is a loopback or private address, which is what a local proxy looks like.
func forwardedFor(r *http.Request, peer string) (string, bool) {
	fwd := r.Header.Get("X-Forwarded-For")
	if fwd == "" {
		return "", false
	}

	peerIP := net.ParseIP(peer)
	if peerIP == nil || (!peerIP.IsLoopback() && !peerIP.IsPrivate()) {
		return "", false
	}

	first := strings.TrimSpace(strings.Split(fwd, ",")[0])
	if net.ParseIP(first) == nil {
		return "", false
	}
	return first, true
}

// WithRateLimit creates a simple IP-based rate limiter middleware.
func WithRateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Clean up old clients every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Static assets and health checks are not worth limiting, and a cold
		// page load fires far more of them than any sane burst allows.
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		// A public peer can send any X-Forwarded-For it likes, so the header is
		// only honoured for the local reverse proxy of a container or host.
		if fwd, ok := forwardedFor(r, ip); ok {
			ip = fwd
		}

		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(RPS, Burst)}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
