package config

import (
	"cmp"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
)

func TestParseFillsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := parse([]byte(`
monitors:
  - name: Site
    url: https://example.com
  - name: Strict
    url: https://api.example.com
    retries: 0
    check_interval: 30
    degraded_threshold: 2000
`))
	require.NoError(t, err)

	site := cfg.Monitors[0]
	assert.EqualValues(t, 60, site.CheckInterval)
	assert.EqualValues(t, 1, site.Retries)
	assert.EqualValues(t, 500, site.DegradedThreshold)

	strict := cfg.Monitors[1]
	assert.EqualValues(t, 0, strict.Retries, "an explicit zero must win over the default")
	assert.EqualValues(t, 30, strict.CheckInterval)
	assert.EqualValues(t, 2000, strict.DegradedThreshold)
}

func TestValidateMonitor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		m       Monitor
		wantErr string
	}{
		{"http", Monitor{URL: "http://example.com"}, ""},
		{"https", Monitor{URL: "https://example.com"}, ""},
		{"tcp with port", Monitor{URL: "tcp://db.example.com:5432"}, ""},
		{"tcp without port", Monitor{URL: "tcp://db.example.com"}, "tcp url must include a port"},
		{"ssl", Monitor{URL: "ssl://example.com"}, ""},
		{"dns", Monitor{URL: "dns://example.com?server=192.168.1.2"}, ""},
		{"ping", Monitor{URL: "ping://192.168.1.1"}, ""},
		{"push", Monitor{URL: "push://s3cret"}, ""},
		{"unsupported scheme", Monitor{URL: "ftp://example.com"}, "scheme must be one of"},
		{"missing url", Monitor{}, "url is required"},
		{"interval too short", Monitor{URL: "https://example.com", CheckInterval: 5}, "check_interval"},
		{"too many retries", Monitor{URL: "https://example.com", Retries: 11}, "retries"},
		{"bad status", Monitor{URL: "https://example.com", ExpectedStatus: 42}, "expected_status"},
		{"keyword on tcp", Monitor{URL: "tcp://db:5432", Keyword: "ok"}, "only work with http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := tt.m
			m.Name = "test"
			m.CheckInterval = cmp.Or(m.CheckInterval, 60)
			m.DegradedThreshold = 500

			err := validateMonitor(m)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateRejectsDuplicatesAndBadWebhooks(t *testing.T) {
	t.Parallel()

	_, err := parse([]byte("monitors:\n  - {name: A, url: https://a.test}\n  - {name: A, url: https://b.test}\n"))
	require.ErrorContains(t, err, "duplicate name")

	_, err = parse([]byte("monitors:\n  - {name: A, url: https://a.test}\n  - {name: B, url: https://a.test}\n"))
	require.ErrorContains(t, err, "already used")

	_, err = parse([]byte("webhooks:\n  - url: ftp://hooks.test\n"))
	require.ErrorContains(t, err, "webhook #1")
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	assert.Empty(t, cfg.Monitors)
}

func TestWatchAppliesChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("monitors: []\n"), 0o600))

	applied := make(chan *Config, 1)
	go Watch(t.Context(), path, func(cfg *Config) error {
		applied <- cfg
		return nil
	})

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("monitors:\n  - {name: A, url: https://a.test}\n"), 0o600))

	select {
	case cfg := <-applied:
		require.Len(t, cfg.Monitors, 1)
	case <-time.After(3 * watchInterval):
		t.Fatal("the changed config was never applied")
	}
}

func TestSyncMatchesMonitorsByName(t *testing.T) {
	t.Parallel()

	q, err := db.Open(t.Context(), t.TempDir())
	require.NoError(t, err)
	ctx := t.Context()

	first, err := Sync(ctx, q, []Monitor{
		{Name: "API", URL: "https://api.test", CheckInterval: 60, Retries: 1, DegradedThreshold: 500},
		{Name: "Old", URL: "https://old.test", CheckInterval: 60, Retries: 1, DegradedThreshold: 500},
	})
	require.NoError(t, err)
	require.Len(t, first, 2)

	second, err := Sync(ctx, q, []Monitor{
		{Name: "API", URL: "tcp://api.test:443", Group: "Core", CheckInterval: 30, DegradedThreshold: 500},
	})
	require.NoError(t, err)
	require.Len(t, second, 1, "monitors missing from the config are removed")

	api := second[0]
	assert.Equal(t, first[0].ID, api.ID, "a changed url keeps the monitor and its history")
	assert.Equal(t, "tcp://api.test:443", api.Url)
	assert.Equal(t, checker.TypeTCP, api.Type)
	assert.Equal(t, "Core", api.GroupName)
	assert.EqualValues(t, 30, api.CheckInterval)
}
