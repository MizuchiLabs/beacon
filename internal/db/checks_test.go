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

func TestGetMonitorStatsCountsChecksInWindow(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	monitor := createTestMonitor(t, q, "https://idle.test")

	checks := []struct {
		at int64
		up bool
	}{
		{100, true},
		{200, true},
		{300, false},
		{400, false},
	}
	for _, c := range checks {
		require.NoError(t, q.UpsertCheck(ctx, &UpsertCheckParams{
			MonitorID:    monitor.ID,
			StatusCode:   200,
			ResponseTime: 42,
			IsUp:         c.up,
			CheckedAt:    c.at,
		}))
	}

	stats, err := q.GetMonitorStats(ctx, 0)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.EqualValues(t, 4, stats[0].CheckCount)
	require.InDelta(t, 50.0, stats[0].UptimePct, 0.001)
	require.EqualValues(t, 42, stats[0].AvgResponseTime)

	// Checks before the window start are excluded, which is what makes a
	// monitor with no history distinguishable from a healthy one.
	stats, err = q.GetMonitorStats(ctx, 201)
	require.NoError(t, err)
	require.EqualValues(t, 2, stats[0].CheckCount)
	require.InDelta(t, 0.0, stats[0].UptimePct, 0.001)
}

func TestGetMonitorStatsWithoutChecks(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	createTestMonitor(t, q, "https://new.test")

	stats, err := q.GetMonitorStats(ctx, 0)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Zero(t, stats[0].CheckCount)
	require.InDelta(t, 0.0, stats[0].UptimePct, 0.001, "a monitor without checks is not 100% up")
}

func TestGetLatestCheckStates(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	up := createTestMonitor(t, q, "https://up.test")
	down := createTestMonitor(t, q, "https://down.test")

	require.NoError(t, q.UpsertCheck(ctx, &UpsertCheckParams{
		MonitorID: up.ID,
		IsUp:      true,
		CheckedAt: 100,
	}))
	require.NoError(t, q.UpsertCheck(ctx, &UpsertCheckParams{
		MonitorID: up.ID,
		IsUp:      false,
		CheckedAt: 200,
	}))
	require.NoError(t, q.UpsertCheck(ctx, &UpsertCheckParams{
		MonitorID: up.ID,
		IsUp:      true,
		CheckedAt: 300,
	}))
	require.NoError(t, q.UpsertCheck(ctx, &UpsertCheckParams{
		MonitorID: down.ID,
		IsUp:      false,
		CheckedAt: 150,
	}))

	states, err := q.GetLatestCheckStates(ctx)
	require.NoError(t, err)
	require.Len(t, states, 2)

	got := make(map[int64]bool, len(states))
	for _, state := range states {
		got[state.MonitorID] = state.IsUp
	}
	require.Equal(t, map[int64]bool{up.ID: true, down.ID: false}, got)
}
