package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestQueries(t *testing.T) *Queries {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", "file::memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, migrate(t.Context(), sqlDB))
	return New(sqlDB)
}

func createTestMonitor(t *testing.T, q *Queries, url string) *Monitor {
	t.Helper()

	monitor, err := q.CreateMonitor(t.Context(), &CreateMonitorParams{
		Name:          url,
		Url:           url,
		CheckInterval: 60,
	})
	require.NoError(t, err)
	return monitor
}

func storeCheck(t *testing.T, q *Queries, monitorID int64, at int64, up bool, ms int64) {
	t.Helper()

	require.NoError(t, q.UpsertCheck(t.Context(), &UpsertCheckParams{
		MonitorID:    monitorID,
		StatusCode:   200,
		ResponseTime: ms,
		IsUp:         up,
		CheckedAt:    at,
	}))
}

func TestGetCheckWindowBucketsCounters(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	monitor := createTestMonitor(t, q, "https://window.test")

	storeCheck(t, q, monitor.ID, 60, true, 30)
	storeCheck(t, q, monitor.ID, 90, true, 120)
	storeCheck(t, q, monitor.ID, 120, false, 0)
	storeCheck(t, q, monitor.ID, 180, true, 900)
	storeCheck(t, q, monitor.ID, 185, true, 6000)

	rows, err := q.GetCheckWindow(ctx, &GetCheckWindowParams{Step: 60, DegradedThreshold: 500, FromTs: 0})
	require.NoError(t, err)
	require.Len(t, rows, 3)

	first := rows[0]
	require.EqualValues(t, 60, first.Ts)
	require.EqualValues(t, 2, first.Total)
	require.EqualValues(t, 2, first.Up)
	require.Zero(t, first.Degraded)
	require.Zero(t, first.Down)
	require.EqualValues(t, 150, first.SumMs)

	down := rows[1]
	require.EqualValues(t, 120, down.Ts)
	require.EqualValues(t, 1, down.Total)
	require.EqualValues(t, 1, down.Down)

	// Slow checks are up, so they count as degraded, not down.
	slow := rows[2]
	require.EqualValues(t, 180, slow.Ts)
	require.EqualValues(t, 2, slow.Total)
	require.Zero(t, slow.Up)
	require.EqualValues(t, 2, slow.Degraded)
	require.EqualValues(t, 6900, slow.SumMs)
}

func TestGetCheckWindowCoarserStep(t *testing.T) {
	q := newTestQueries(t)
	monitor := createTestMonitor(t, q, "https://step.test")

	storeCheck(t, q, monitor.ID, 60, true, 30)
	storeCheck(t, q, monitor.ID, 290, false, 0)

	rows, err := q.GetCheckWindow(t.Context(), &GetCheckWindowParams{Step: 300, DegradedThreshold: 500, FromTs: 0})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 0, rows[0].Ts)
	require.EqualValues(t, 2, rows[0].Total)
	require.EqualValues(t, 1, rows[0].Up)
	require.EqualValues(t, 1, rows[0].Down)
}

func TestGetCheckResponseTimesSortsUpChecks(t *testing.T) {
	q := newTestQueries(t)
	monitor := createTestMonitor(t, q, "https://times.test")

	storeCheck(t, q, monitor.ID, 60, true, 900)
	storeCheck(t, q, monitor.ID, 90, false, 10)
	storeCheck(t, q, monitor.ID, 120, true, 30)

	times, err := q.GetCheckResponseTimes(t.Context(), 0)
	require.NoError(t, err)
	require.Len(t, times, 2, "down checks are excluded")
	require.Equal(t, monitor.ID, times[0].MonitorID)
	require.EqualValues(t, 30, times[0].ResponseTime)
	require.EqualValues(t, 900, times[1].ResponseTime)
}

func TestGetLatestChecks(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	up := createTestMonitor(t, q, "https://up.test")
	down := createTestMonitor(t, q, "https://down.test")
	quiet := createTestMonitor(t, q, "https://quiet.test")

	storeCheck(t, q, up.ID, 100, true, 20)
	storeCheck(t, q, up.ID, 200, false, 0)
	storeCheck(t, q, up.ID, 300, true, 42)
	storeCheck(t, q, down.ID, 150, false, 0)

	latest, err := q.GetLatestChecks(ctx)
	require.NoError(t, err)
	require.Len(t, latest, 2, "a monitor without checks has no latest check")

	got := make(map[int64]*GetLatestChecksRow, len(latest))
	for _, l := range latest {
		got[l.MonitorID] = l
	}

	require.EqualValues(t, 300, got[up.ID].CheckedAt)
	require.True(t, got[up.ID].IsUp)
	require.EqualValues(t, 42, got[up.ID].ResponseTime)
	require.False(t, got[down.ID].IsUp)
	require.NotContains(t, got, quiet.ID)
}
