package scheduler

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

func newTestService(t *testing.T) (*Service, *db.Queries) {
	t.Helper()

	q, err := db.Open(t.Context(), t.TempDir())
	require.NoError(t, err)
	chk, err := checker.New()
	require.NoError(t, err)
	n, err := notify.New(t.Context(), q)
	require.NoError(t, err)
	s, err := New(q, chk, n)
	require.NoError(t, err)
	return s, q
}

func createMonitor(t *testing.T, q *db.Queries, name, url, typ string) *db.Monitor {
	t.Helper()

	m, err := q.UpsertMonitor(t.Context(), &db.UpsertMonitorParams{
		Name:              name,
		Url:               url,
		Type:              typ,
		CheckInterval:     60,
		DegradedThreshold: 500,
	})
	require.NoError(t, err)
	return m
}

func latest(t *testing.T, q *db.Queries, monitorID int64) *db.GetLatestChecksRow {
	t.Helper()

	rows, err := q.GetLatestChecks(t.Context())
	require.NoError(t, err)
	for _, r := range rows {
		if r.MonitorID == monitorID {
			return r
		}
	}
	return nil
}

func TestTransitionedReportsOnlyChanges(t *testing.T) {
	t.Parallel()

	s := &Service{lastUp: make(map[int64]bool)}

	require.False(t, s.transitioned(1, true), "first observation is not a transition")
	require.False(t, s.transitioned(1, true))
	require.True(t, s.transitioned(1, false), "up to down is a transition")
	require.False(t, s.transitioned(1, false))
	require.True(t, s.transitioned(1, true), "down to up is a transition")
	require.False(t, s.transitioned(2, true), "state is tracked per monitor")
}

func TestCertWarnDueFiresOncePerDayPerMonitor(t *testing.T) {
	t.Parallel()

	s := &Service{lastCertWarn: make(map[int64]int64)}

	require.True(t, s.certWarnDue(1))
	require.False(t, s.certWarnDue(1), "a second check the same day must not warn again")
	require.True(t, s.certWarnDue(2), "warnings are tracked per monitor")
}

func TestCutoffsUseConfiguredRetention(t *testing.T) {
	t.Setenv("BEACON_RETENTION_DAYS", "7")
	t.Setenv("BEACON_HISTORY_DAYS", "90")

	s, err := New(nil, nil, nil)
	require.NoError(t, err)

	checks, rollups := s.cutoffs()
	require.WithinDuration(t, time.Now().AddDate(0, 0, -7), time.Unix(checks, 0), time.Minute)
	require.WithinDuration(t, time.Now().AddDate(0, 0, -90), time.Unix(rollups, 0), time.Minute)
}

func TestHeartbeat(t *testing.T) {
	t.Parallel()

	s, q := newTestService(t)
	m := createMonitor(t, q, "backup", "push://s3cret", checker.TypePush)
	s.Schedule(t.Context(), []*db.Monitor{m})

	require.ErrorIs(t, s.Heartbeat(t.Context(), "wrong", true, ""), ErrUnknownHeartbeat)

	require.NoError(t, s.Heartbeat(t.Context(), "s3cret", true, ""))
	require.True(t, latest(t, q, m.ID).IsUp)

	// Pretend the last ping is long overdue.
	s.mu.Lock()
	s.lastPing[m.ID] = time.Now().Add(-2 * time.Minute)
	s.mu.Unlock()
	time.Sleep(time.Second) // checks are keyed by second

	s.checkHeartbeat(t.Context(), m, time.Minute)
	assert.False(t, latest(t, q, m.ID).IsUp, "a missed heartbeat is down")
}

func TestCheckRetriesBeforeFailing(t *testing.T) { //nolint:paralleltest // swaps retryDelay
	retryDelay = 10 * time.Millisecond
	t.Cleanup(func() { retryDelay = 5 * time.Second })

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	t.Cleanup(server.Close)

	s, q := newTestService(t)
	m := createMonitor(t, q, "flaky", server.URL, checker.TypeHTTP)
	m.Retries = 1

	s.check(t.Context(), m)
	assert.EqualValues(t, 2, calls.Load())
	assert.True(t, latest(t, q, m.ID).IsUp, "one blip that recovers on retry is up")
}
