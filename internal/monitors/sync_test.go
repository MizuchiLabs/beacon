package monitors

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mizuchilabs/beacon/internal/checker"
)

func TestValidateAcceptsKnownSchemes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"http", "http://example.com", ""},
		{"https", "https://example.com", ""},
		{"tcp with port", "tcp://db.example.com:5432", ""},
		{"tcp without port", "tcp://db.example.com", "tcp url must include a port"},
		{"ssl with port", "ssl://example.com:636", ""},
		{"ssl without port", "ssl://example.com", ""},
		{"unsupported scheme", "ftp://example.com", "scheme must be"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validate([]monitor{{Name: "test", URL: tt.url, CheckInterval: 60}})
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMonitorTypeFromScheme(t *testing.T) {
	t.Parallel()

	require.Equal(t, checker.TypeHTTP, monitorType("https://example.com"))
	require.Equal(t, checker.TypeTCP, monitorType("tcp://db.example.com:5432"))
	require.Equal(t, checker.TypeSSL, monitorType("ssl://example.com"))
}
