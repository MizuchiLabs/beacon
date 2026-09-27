package db

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestQueries(t *testing.T) *Queries {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", "file::memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, migrate(t.Context(), sqlDB, ""))
	return New(sqlDB)
}

func createTestMonitor(t *testing.T, q *Queries, url string) *Monitor {
	t.Helper()

	monitor, err := q.UpsertMonitor(t.Context(), &UpsertMonitorParams{
		Name:              url,
		Url:               url,
		CheckInterval:     60,
		DegradedThreshold: 500,
	})
	require.NoError(t, err)
	return monitor
}

func storeCheck(t *testing.T, q *Queries, monitorID int64, at int64, up bool, ms int64) {
	t.Helper()

	require.NoError(t, q.InsertCheck(t.Context(), &InsertCheckParams{
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

	rows, err := q.GetCheckWindow(ctx, &GetCheckWindowParams{Step: 60, FromTs: 0})
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

	rows, err := q.GetCheckWindow(t.Context(), &GetCheckWindowParams{Step: 300, FromTs: 0})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 0, rows[0].Ts)
	require.EqualValues(t, 2, rows[0].Total)
	require.EqualValues(t, 1, rows[0].Up)
	require.EqualValues(t, 1, rows[0].Down)
}

func TestGetResponseTimesSortsUpChecks(t *testing.T) {
	q := newTestQueries(t)
	monitor := createTestMonitor(t, q, "https://times.test")
	other := createTestMonitor(t, q, "https://other.test")

	storeCheck(t, q, monitor.ID, 60, true, 900)
	storeCheck(t, q, monitor.ID, 90, false, 10)
	storeCheck(t, q, monitor.ID, 120, true, 30)
	storeCheck(t, q, other.ID, 120, true, 5)

	times, err := q.GetResponseTimes(t.Context(), &GetResponseTimesParams{MonitorID: monitor.ID, FromTs: 0})
	require.NoError(t, err)
	require.Equal(t, []int64{30, 900}, times, "down checks and other monitors are excluded")
}

func TestTriggerKeepsRollupsInStepWithChecks(t *testing.T) {
	q := newTestQueries(t)
	ctx := t.Context()
	monitor := createTestMonitor(t, q, "https://rollup.test")

	storeCheck(t, q, monitor.ID, 3600, true, 100)
	storeCheck(t, q, monitor.ID, 3660, true, 900)
	storeCheck(t, q, monitor.ID, 3720, false, 30000)
	storeCheck(t, q, monitor.ID, 3720, true, 1) // same second, ignored
	storeCheck(t, q, monitor.ID, 7200, true, 50)

	rollups, err := q.GetRollupWindow(ctx, &GetRollupWindowParams{Step: 3600, FromTs: 0})
	require.NoError(t, err)
	require.Len(t, rollups, 2)
	assert.Equal(
		t,
		GetRollupWindowRow{MonitorID: monitor.ID, Ts: 3600, Total: 3, Up: 1, Degraded: 1, Down: 1, SumMs: 1000},
		*rollups[0],
	)
	assert.Equal(t, GetRollupWindowRow{MonitorID: monitor.ID, Ts: 7200, Total: 1, Up: 1, SumMs: 50}, *rollups[1])

	raw, err := q.GetCheckWindow(ctx, &GetCheckWindowParams{Step: 3600, FromTs: 0})
	require.NoError(t, err)
	for i := range raw {
		assert.Equal(t, GetCheckWindowRow(*rollups[i]), *raw[i], "rollups and raw checks must agree")
	}

	uptime, err := q.GetUptime(ctx, &GetUptimeParams{MonitorID: monitor.ID, FromTs: 0})
	require.NoError(t, err)
	assert.Equal(t, GetUptimeRow{Total: 4, Alive: 3}, *uptime)
}

func TestMigrateFromFirstSchemaKeepsDataAndBackfills(t *testing.T) {
	ctx := t.Context()
	sqlDB, err := sql.Open("sqlite", "file::memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	old, err := os.ReadFile("testdata/v1_schema.sql")
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, string(old))
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO monitors (id, name, url, check_interval) VALUES (7, 'API', 'https://api.test', 60);
		INSERT INTO checks (monitor_id, status_code, response_time, is_up, checked_at)
		VALUES (7, 200, 40, 1, 3600), (7, 0, 0, 0, 3660);
		INSERT INTO push_subscriptions (monitor_id, endpoint, p256dh_key, auth_key) VALUES (7, 'https://push.test', 'k', 'a');`)
	require.NoError(t, err)

	require.NoError(t, migrate(ctx, sqlDB, ""))
	q := New(sqlDB)
	require.NoError(t, q.BackfillRollups(ctx))

	monitors, err := q.GetMonitors(ctx)
	require.NoError(t, err)
	require.Len(t, monitors, 1)
	assert.EqualValues(t, 7, monitors[0].ID, "ids survive so checks and subscriptions stay attached")
	assert.EqualValues(t, 1, monitors[0].Retries)
	assert.EqualValues(t, 500, monitors[0].DegradedThreshold)

	ids, err := q.GetSubscribedMonitorIDs(ctx, "https://push.test")
	require.NoError(t, err)
	assert.Equal(t, []int64{7}, ids)

	rollups, err := q.GetRollupWindow(ctx, &GetRollupWindowParams{Step: 3600, FromTs: 0})
	require.NoError(t, err)
	require.Len(t, rollups, 1)
	assert.EqualValues(t, 2, rollups[0].Total, "existing checks are backfilled into rollups")

	require.NoError(t, q.BackfillRollups(ctx))
	rollups, err = q.GetRollupWindow(ctx, &GetRollupWindowParams{Step: 3600, FromTs: 0})
	require.NoError(t, err)
	assert.EqualValues(t, 2, rollups[0].Total, "a second backfill must not double count")
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
