package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForwardedFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		peer  string
		xfwd  string
		want  string
		trust bool
	}{
		{"proxy peer", "172.17.0.1", "203.0.113.7", "203.0.113.7", true},
		{"loopback peer", "127.0.0.1", "203.0.113.7", "203.0.113.7", true},
		{"first hop wins", "10.0.0.5", "203.0.113.7, 10.0.0.1", "203.0.113.7", true},
		{"public peer", "198.51.100.9", "203.0.113.7", "", false},
		{"garbage value", "127.0.0.1", "not-an-ip", "", false},
		{"missing header", "127.0.0.1", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/api/monitors", nil)
			if tt.xfwd != "" {
				r.Header.Set("X-Forwarded-For", tt.xfwd)
			}

			got, ok := forwardedFor(r, tt.peer)
			require.Equal(t, tt.trust, ok)
			require.Equal(t, tt.want, got)
		})
	}
}
