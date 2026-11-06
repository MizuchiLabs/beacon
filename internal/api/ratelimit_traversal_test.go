package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// A path that only starts with /api/ after cleaning must count against the
// same client bucket as the plain path.
func TestRateLimitTraversalBypass(t *testing.T) {
	t.Parallel()

	s, err := New(nil, nil, nil)
	require.NoError(t, err)

	do := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.21:443"
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		return rec.Code
	}

	for range 100 {
		require.Equal(t, http.StatusOK, do("/api/config"))
	}
	require.Equal(t, http.StatusTooManyRequests, do("/api/config"))
	require.Equal(t, http.StatusTooManyRequests, do("/x/../api/config"))
}
