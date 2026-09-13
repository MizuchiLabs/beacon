package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mizuchilabs/beacon/internal/db"
)

func TestPercentileAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		sorted []int64
		p      int
		want   int64
	}{
		{"single sample", []int64{42}, 50, 42},
		{"median of an even sample rounds up", []int64{10, 20, 30, 40}, 50, 20},
		{"p99 of a tight sample", []int64{10, 20, 30, 40}, 99, 40},
		{"p99 takes the tail", []int64{10, 20, 30, 40, 50, 5000}, 99, 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, percentileAt(tt.sorted, tt.p))
		})
	}
}

func TestStepForWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seconds int64
		want    int64
	}{
		{"one hour", 3600, 300},
		{"one day", 86400, 1800},
		{"one week", 604800, 14400},
		{"two weeks", 1209600, 43200},
		{"one month", 2592000, 86400},
		{"larger than the ladder", 31536000, 86400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, stepForWindow(tt.seconds))
		})
	}
}

func TestStepForWindowStaysUnderPointCap(t *testing.T) {
	t.Parallel()

	for _, seconds := range []int64{60, 3600, 86400, 604800, 1209600, 2592000} {
		step := stepForWindow(seconds)
		require.LessOrEqual(t, seconds/step, int64(maxPoints), "window %ds", seconds)
		require.Zero(t, 86400%step, "step %ds does not divide a day", step)
	}
}

func TestStatusOf(t *testing.T) {
	t.Parallel()

	const now = 1_000_000

	tests := []struct {
		name     string
		last     *db.GetLatestChecksRow
		interval int64
		want     string
	}{
		{"no checks yet", nil, 60, statusUnknown},
		{
			"fresh and up",
			&db.GetLatestChecksRow{IsUp: true, ResponseTime: 42, CheckedAt: now - 60},
			60,
			statusOperational,
		},
		{
			"fresh and slow",
			&db.GetLatestChecksRow{IsUp: true, ResponseTime: 501, CheckedAt: now - 60},
			60,
			statusDegraded,
		},
		{"down", &db.GetLatestChecksRow{IsUp: false, CheckedAt: now - 60}, 60, statusDown},
		{"stale", &db.GetLatestChecksRow{IsUp: true, CheckedAt: now - 181}, 60, statusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, statusOf(tt.last, tt.interval, now))
		})
	}
}

func TestBuildPointsFillsGaps(t *testing.T) {
	t.Parallel()

	m := &db.Monitor{ID: 1, CheckInterval: 50, CreatedAt: 0}
	rows := []*db.GetCheckWindowRow{
		{MonitorID: 1, Ts: 100, Total: 4, Up: 3, Down: 1, SumMs: 400},
	}

	points := buildPoints(m, rows, 0, 300, 100)
	require.Len(t, points, 4)

	require.Equal(t, time.Unix(0, 0), points[0].Timestamp)
	require.False(t, points[0].HasData, "a bucket without checks must not report as up")
	require.Zero(t, points[0].Total)
	require.Nil(t, points[0].AvgMs)
	require.EqualValues(t, 2, points[0].Expected)

	require.True(t, points[1].HasData)
	require.EqualValues(t, 3, points[1].Up)
	require.EqualValues(t, 1, points[1].Down)
	require.EqualValues(t, 100, *points[1].AvgMs)

	require.False(t, points[3].HasData)
}

func TestBuildPointsExpectations(t *testing.T) {
	t.Parallel()

	t.Run("every bucket expects checks when it spans the interval", func(t *testing.T) {
		t.Parallel()
		m := &db.Monitor{ID: 1, CheckInterval: 60, CreatedAt: 0}
		points := buildPoints(m, nil, 0, 300, 100)
		for _, p := range points {
			require.EqualValues(t, 1, p.Expected)
		}
	})

	t.Run("a bucket shorter than the check interval expects nothing", func(t *testing.T) {
		t.Parallel()
		m := &db.Monitor{ID: 1, CheckInterval: 200, CreatedAt: 0}
		points := buildPoints(m, nil, 0, 100, 100)
		require.Zero(t, points[0].Expected)
	})
}

func TestStatsCountDegradedChecksAsUptime(t *testing.T) {
	t.Parallel()

	m := &db.Monitor{ID: 1, CheckInterval: 60, CreatedAt: 0}
	rows := []*db.GetCheckWindowRow{
		{MonitorID: 1, Ts: 60, Total: 2, Degraded: 2, SumMs: 2000},
	}

	got := stats(m, rows, nil, []int64{1000, 1000}, 60, 60, 60)
	require.InDelta(t, 100, *got.UptimePct, 0.001)
	require.EqualValues(t, 1000, *got.AvgResponseTime)
	require.NotNil(t, got.Percentiles, "degraded checks still feed the percentiles")
	require.Zero(t, got.Datapoints[0].Up)
	require.EqualValues(t, 2, got.Datapoints[0].Degraded)
}

func TestStatsWithoutChecksReportsNoNumbers(t *testing.T) {
	t.Parallel()

	m := &db.Monitor{ID: 1, CheckInterval: 60, CreatedAt: 0}
	got := stats(m, nil, nil, nil, 0, 60, 60)

	require.Equal(t, statusUnknown, got.Status)
	require.Nil(t, got.UptimePct)
	require.Nil(t, got.AvgResponseTime)
	require.Nil(t, got.Percentiles)
	require.False(t, got.Datapoints[0].HasData)
}
