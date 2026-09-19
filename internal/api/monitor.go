package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
)

const (
	// Status reported for the latest check of a monitor.
	statusOperational = "operational"
	statusDegraded    = "degraded"
	statusDown        = "down"
	statusUnknown     = "unknown"
	// maxPoints caps how many buckets a window is split into.
	maxPoints = 50
	// slowThresh is the response time above which an up check counts as degraded.
	slowThresh = 500 * time.Millisecond
	// stalenessFactor is how many missed intervals make a monitor unknown.
	stalenessFactor = 3
)

// stepLadder are the bucket sizes in seconds a window may be reduced to, from
// one minute up to a day. Nothing here depends on how the chart is drawn.
var stepLadder = []int64{60, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400}

// MonitorStats is the aggregated uptime and latency stats for one monitor.
// Uptime, latency and percentiles are null when the window holds no checks.
type MonitorStats struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	URL             string       `json:"url"`
	Type            string       `json:"type"                      doc:"What kind of check this monitor runs"                                enum:"http,tcp,ssl"`
	CheckInterval   int64        `json:"check_interval"`
	Status          string       `json:"status"                    doc:"Status of the latest check, unknown when the monitor has gone quiet" enum:"operational,degraded,down,unknown"`
	DaysRemaining   *int64       `json:"days_remaining"            doc:"Days until the certificate expires, https and ssl monitors only"`
	IgnoreCert      bool         `json:"ignore_cert_expiry"        doc:"Whether certificate expiry is ignored for status and warnings"`
	LastCheckedAt   *time.Time   `json:"last_checked_at,omitempty" doc:"Timestamp of the latest check"`
	AvgResponseTime *int64       `json:"avg_response_time"`
	UptimePct       *float64     `json:"uptime_pct"`
	Percentiles     *Percentiles `json:"percentiles,omitempty"`
	Datapoints      []DataPoint  `json:"data_points"`
}

// Percentiles are response time percentiles in milliseconds.
type Percentiles struct {
	P50 int64 `json:"p50"`
	P75 int64 `json:"p75"`
	P90 int64 `json:"p90"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
}

// DataPoint is one bucket of check results. Every chart consumes this same
// shape, ratios are derived from the counters by the renderer.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	HasData   bool      `json:"has_data"  doc:"Whether any check landed in this bucket"`
	Expected  int64     `json:"expected"  doc:"Checks this bucket should hold, zero when it is shorter than the check interval. No data with a non zero expectation is a gap in monitoring"`
	Total     int64     `json:"total"`
	Up        int64     `json:"up"`
	Degraded  int64     `json:"degraded"`
	Down      int64     `json:"down"`
	AvgMs     *int64    `json:"avg_ms"    doc:"Mean response time of every check in the bucket"`
}

type GetMonitorsInput struct {
	Seconds int64 `query:"seconds" default:"86400" minimum:"60" maximum:"31536000" doc:"How far back to aggregate, in seconds"`
}

type MonitorsOutput struct {
	Body []MonitorStats
}

type MonitorService struct {
	q *db.Queries
}

func NewMonitorService(api huma.API, q *db.Queries) *MonitorService {
	svc := &MonitorService{q: q}
	huma.Register(api, huma.Operation{
		OperationID: "get-monitors",
		Method:      http.MethodGet,
		Path:        "/api/monitors",
		Summary:     "Get monitor stats",
		Description: "Aggregated uptime and response time stats per monitor over the requested window.",
		Tags:        []string{"Monitors"},
	}, svc.getMonitors)
	return svc
}

func (s *MonitorService) getMonitors(
	ctx context.Context,
	in *GetMonitorsInput,
) (*MonitorsOutput, error) {
	now := time.Now()
	step := stepForWindow(in.Seconds)
	since := now.Unix() - in.Seconds
	since -= since % step

	monitors, err := s.q.GetMonitors(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get monitors")
	}

	latest, err := s.q.GetLatestChecks(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get latest checks")
	}
	latestByMonitor := make(map[int64]*db.GetLatestChecksRow, len(latest))
	for _, l := range latest {
		latestByMonitor[l.MonitorID] = l
	}

	// Two bounded reads of the raw checks. The cost scales with the checks in
	// the window, the step only shapes how they are grouped.
	rows, err := s.q.GetCheckWindow(ctx, &db.GetCheckWindowParams{
		Step:              step,
		DegradedThreshold: slowThresh.Milliseconds(),
		FromTs:            since,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to aggregate checks")
	}
	pointsByMonitor := make(map[int64][]*db.GetCheckWindowRow, len(monitors))
	for _, r := range rows {
		pointsByMonitor[r.MonitorID] = append(pointsByMonitor[r.MonitorID], r)
	}
	timesByMonitor := make(map[int64][]int64, len(monitors))
	times, err := s.q.GetCheckResponseTimes(ctx, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load response times")
	}
	for _, t := range times {
		timesByMonitor[t.MonitorID] = append(timesByMonitor[t.MonitorID], t.ResponseTime)
	}

	result := make([]MonitorStats, len(monitors))
	for i, m := range monitors {
		result[i] = stats(
			m,
			pointsByMonitor[m.ID],
			latestByMonitor[m.ID],
			timesByMonitor[m.ID],
			since,
			now.Unix(),
			step,
		)
	}
	return &MonitorsOutput{Body: result}, nil
}

// stats folds the window totals out of the same grouped rows the chart draws,
// so the numbers above the chart and the bars inside it can never disagree.
func stats(
	m *db.Monitor,
	rows []*db.GetCheckWindowRow,
	last *db.GetLatestChecksRow,
	times []int64,
	since int64,
	now int64,
	step int64,
) MonitorStats {
	row := MonitorStats{
		ID:            m.ID,
		Name:          m.Name,
		URL:           m.Url,
		CheckInterval: m.CheckInterval,
		IgnoreCert:    m.IgnoreCertExpiry,
		Type:          m.Type,
		Status:        statusOf(m, last, now),
		Datapoints:    buildPoints(m, rows, since, now, step),
	}
	if last != nil {
		row.LastCheckedAt = new(time.Unix(last.CheckedAt, 0))
		row.DaysRemaining = last.DaysRemaining
	}

	var total, up, degraded, sum int64
	for _, r := range rows {
		total += r.Total
		up += r.Up
		degraded += r.Degraded
		sum += r.SumMs
	}

	// Degraded checks are still up, so they count towards uptime and make up
	// part of the percentile population along with the fast ones.
	alive := up + degraded
	if total > 0 {
		row.UptimePct = new(float64(alive) / float64(total) * 100)
		row.AvgResponseTime = new(sum / total)
		if len(times) > 0 {
			row.Percentiles = percentiles(times)
		}
	}
	return row
}

// buildPoints walks the window on step boundaries and fills the holes the
// aggregation left behind. A bucket without checks has to show up as a bucket
// without checks, otherwise a gap in the history looks like a healthy one.
func buildPoints(
	m *db.Monitor,
	rows []*db.GetCheckWindowRow,
	since int64,
	now int64,
	step int64,
) []DataPoint {
	byTS := make(map[int64]*db.GetCheckWindowRow, len(rows))
	for _, r := range rows {
		byTS[r.Ts] = r
	}

	points := make([]DataPoint, 0, (now-since)/step+1)
	for ts := since; ts <= now; ts += step {
		point := DataPoint{Timestamp: time.Unix(ts, 0)}
		// A bucket shorter than the check interval is empty by construction, so
		// nothing can be missing from it.
		if step >= m.CheckInterval {
			point.Expected = step / m.CheckInterval
		}

		r, ok := byTS[ts]
		if !ok {
			points = append(points, point)
			continue
		}

		point.HasData = r.Total > 0
		point.Total = r.Total
		point.Up = r.Up
		point.Degraded = r.Degraded
		point.Down = r.Down
		if r.Total > 0 {
			point.AvgMs = new(r.SumMs / r.Total)
		}
		points = append(points, point)
	}
	return points
}

// statusOf reports the latest check, not the window average. A monitor that
// just died has a healthy looking uptime, and one that stopped reporting is
// unknown rather than up.
func statusOf(m *db.Monitor, last *db.GetLatestChecksRow, now int64) string {
	if last == nil || now-last.CheckedAt > stalenessFactor*m.CheckInterval {
		return statusUnknown
	}
	if !last.IsUp {
		return statusDown
	}
	if last.ResponseTime > slowThresh.Milliseconds() {
		return statusDegraded
	}
	if !m.IgnoreCertExpiry && last.DaysRemaining != nil && *last.DaysRemaining < checker.CertWarnDays {
		return statusDegraded
	}
	return statusOperational
}

func stepForWindow(seconds int64) int64 {
	for _, step := range stepLadder {
		if seconds <= step*maxPoints {
			return step
		}
	}
	return stepLadder[len(stepLadder)-1]
}

// percentiles returns nearest-rank percentiles of a sorted latency sample.
func percentiles(sorted []int64) *Percentiles {
	return &Percentiles{
		P50: percentileAt(sorted, 50),
		P75: percentileAt(sorted, 75),
		P90: percentileAt(sorted, 90),
		P95: percentileAt(sorted, 95),
		P99: percentileAt(sorted, 99),
	}
}

// percentileAt returns the nearest-rank percentile of an ascending sample.
func percentileAt(sorted []int64, p int) int64 {
	rank := (int64(p)*int64(len(sorted)) + 99) / 100
	return sorted[rank-1]
}
