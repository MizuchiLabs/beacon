package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitAPIScoped(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimitAPI(3, time.Minute)(next)

	get := func(path, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = ip + ":443"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for range 3 {
		require.Equal(t, http.StatusOK, get("/api/monitors", "192.0.2.11").Code)
	}
	require.Equal(t, http.StatusTooManyRequests, get("/api/monitors", "192.0.2.11").Code)

	// A second client keeps its own bucket.
	require.Equal(t, http.StatusOK, get("/api/monitors", "192.0.2.12").Code)

	// Static assets and health probes never touch the bucket.
	require.Equal(t, http.StatusOK, get("/healthz", "192.0.2.11").Code)
	require.Equal(t, http.StatusOK, get("/_app/immutable/app.js", "192.0.2.11").Code)
}

func TestRateLimitUsesProxyHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	t.Setenv("BEACON_TRUSTED_PROXIES", "nginx")
	handler := rateLimitAPI(2, time.Minute)(next)

	get := func(ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/incidents", nil)
		req.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for range 2 {
		require.Equal(t, http.StatusOK, get("203.0.113.7").Code)
	}
	require.Equal(t, http.StatusTooManyRequests, get("203.0.113.7").Code)

	// Different client behind the same proxy address gets its own bucket.
	require.Equal(t, http.StatusOK, get("203.0.113.8").Code)
}
