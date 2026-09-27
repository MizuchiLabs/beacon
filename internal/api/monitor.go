package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

const (
	// Status reported for the latest check of a monitor.
	statusOperational = "operational"
	statusDegraded    = "degraded"
	statusDown        = "down"
	statusUnknown     = "unknown"
	// maxPoints caps how many buckets a window is split into.
	maxPoints = 50
	// stalenessFactor is how many missed intervals make a monitor unknown.
	stalenessFactor = 3
	// rollupStep is the bucket size of the check_rollups table. Windows with a
	// step at least this large are read from the rollups.
	rollupStep = 3600
)

// stepLadder are the bucket sizes in seconds a window may be reduced to, from
// one minute up to a week. Nothing here depends on how the chart is drawn.
var stepLadder = []int64{60, 300, 900, 1800, 3600, 7200, 14400, 43200, 86400, 172800, 604800}

// MonitorStats is the aggregated uptime and latency stats for one monitor.
// Uptime and latency are null when the window holds no checks.
type MonitorStats struct {
	ID                int64       `json:"id"`
	Name              string      `json:"name"`
	URL               string      `json:"url"                       doc:"Monitor url, empty for push monitors so the token stays private"`
	Type              string      `json:"type"                      doc:"What kind of check this monitor runs"                                enum:"http,tcp,ssl,dns,ping,push"`
	Group             string      `json:"group"                     doc:"Group the monitor is shown under, empty for none"`
	CheckInterval     int64       `json:"check_interval"`
	DegradedThreshold int64       `json:"degraded_threshold"        doc:"Response time in ms above which an up check counts as degraded"`
	Status            string      `json:"status"                    doc:"Status of the latest check, unknown when the monitor has gone quiet" enum:"operational,degraded,down,unknown"`
	DaysRemaining     *int64      `json:"days_remaining"            doc:"Days until the certificate expires, https and ssl monitors only"`
	IgnoreCert        bool        `json:"ignore_cert_expiry"        doc:"Whether certificate expiry is ignored for status and warnings"`
	LastCheckedAt     *time.Time  `json:"last_checked_at,omitempty" doc:"Timestamp of the latest check"`
	AvgResponseTime   *int64      `json:"avg_response_time"         doc:"Mean response time of the up checks in the window"`
	UptimePct         *float64    `json:"uptime_pct"`
	Datapoints        []DataPoint `json:"data_points"`
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
	AvgMs     *int64    `json:"avg_ms"    doc:"Mean response time of the up checks in the bucket"`
}

// Percentiles are response time percentiles of the up checks in milliseconds.
type Percentiles struct {
	P50 int64 `json:"p50"`
	P75 int64 `json:"p75"`
	P90 int64 `json:"p90"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
}

type WindowInput struct {
	Seconds int64 `query:"seconds" default:"86400" minimum:"60" maximum:"31536000" doc:"How far back to aggregate, in seconds"`
}

type MonitorsOutput struct {
	Body []MonitorStats
}

type PercentilesInput struct {
	WindowInput

	ID int64 `path:"id" doc:"Monitor ID"`
}

type PercentilesOutput struct {
	Body struct {
		Percentiles *Percentiles `json:"percentiles" doc:"Null when the window holds no up checks"`
	}
}

type monitorHandlers struct {
	q *db.Queries
}

func registerMonitors(api huma.API, q *db.Queries) {
	h := &monitorHandlers{q: q}
	huma.Register(api, huma.Operation{
		OperationID: "get-monitors",
		Method:      http.MethodGet,
		Path:        "/api/monitors",
		Summary:     "Get monitor stats",
		Description: "Aggregated uptime and response time stats per monitor over the requested window.",
		Tags:        []string{"Monitors"},
	}, h.getMonitors)
	huma.Register(api, huma.Operation{
		OperationID: "get-monitor-percentiles",
		Method:      http.MethodGet,
		Path:        "/api/monitors/{id}/percentiles",
		Summary:     "Get response time percentiles",
		Description: "Percentiles of the raw checks in the window, null without checks. Raw checks only go back as far as BEACON_RETENTION_DAYS.",
		Tags:        []string{"Monitors"},
	}, h.getPercentiles)
}

func (h *monitorHandlers) getMonitors(ctx context.Context, in *WindowInput) (*MonitorsOutput, error) {
	now := time.Now().Unix()
	step := stepForWindow(in.Seconds)
	since := now - in.Seconds
	since -= since % step

	monitors, err := h.q.GetMonitors(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get monitors")
	}

	latest, err := h.q.GetLatestChecks(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get latest checks")
	}
	latestByMonitor := make(map[int64]*db.GetLatestChecksRow, len(latest))
	for _, l := range latest {
		latestByMonitor[l.MonitorID] = l
	}

	rows, err := h.window(ctx, step, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to aggregate checks")
	}
	rowsByMonitor := make(map[int64][]*db.GetCheckWindowRow, len(monitors))
	for _, r := range rows {
		rowsByMonitor[r.MonitorID] = append(rowsByMonitor[r.MonitorID], r)
	}

	result := make([]MonitorStats, len(monitors))
	for i, m := range monitors {
		result[i] = stats(m, rowsByMonitor[m.ID], latestByMonitor[m.ID], since, now, step)
	}
	return &MonitorsOutput{Body: result}, nil
}

// window groups the checks since from into buckets of step seconds. Short
// windows need the raw checks, long ones read the much smaller hourly rollups.
func (h *monitorHandlers) window(ctx context.Context, step, from int64) ([]*db.GetCheckWindowRow, error) {
	if step < rollupStep {
		return h.q.GetCheckWindow(ctx, &db.GetCheckWindowParams{Step: step, FromTs: from})
	}

	rollups, err := h.q.GetRollupWindow(ctx, &db.GetRollupWindowParams{Step: step, FromTs: from})
	if err != nil {
		return nil, err
	}
	rows := make([]*db.GetCheckWindowRow, len(rollups))
	for i, r := range rollups {
		rows[i] = new(db.GetCheckWindowRow(*r))
	}
	return rows, nil
}

func (h *monitorHandlers) getPercentiles(ctx context.Context, in *PercentilesInput) (*PercentilesOutput, error) {
	times, err := h.q.GetResponseTimes(ctx, &db.GetResponseTimesParams{
		MonitorID: in.ID,
		FromTs:    time.Now().Unix() - in.Seconds,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load response times")
	}

	out := &PercentilesOutput{}
	if len(times) > 0 {
		out.Body.Percentiles = &Percentiles{
			P50: percentileAt(times, 50),
			P75: percentileAt(times, 75),
			P90: percentileAt(times, 90),
			P95: percentileAt(times, 95),
			P99: percentileAt(times, 99),
		}
	}
	return out, nil
}

// stats folds the window totals out of the same grouped rows the chart draws,
// so the numbers above the chart and the bars inside it can never disagree.
func stats(
	m *db.Monitor,
	rows []*db.GetCheckWindowRow,
	last *db.GetLatestChecksRow,
	since, now, step int64,
) MonitorStats {
	row := MonitorStats{
		ID:                m.ID,
		Name:              m.Name,
		URL:               notify.PublicURL(m),
		Type:              m.Type,
		Group:             m.GroupName,
		CheckInterval:     m.CheckInterval,
		DegradedThreshold: m.DegradedThreshold,
		IgnoreCert:        m.IgnoreCertExpiry,
		Status:            statusOf(m, last, now),
		Datapoints:        buildPoints(m, rows, since, now, step),
	}
	if last != nil {
		row.LastCheckedAt = new(time.Unix(last.CheckedAt, 0))
		row.DaysRemaining = last.DaysRemaining
	}

	var total, alive, sum int64
	for _, r := range rows {
		total += r.Total
		alive += r.Up + r.Degraded
		sum += r.SumMs
	}

	// Degraded checks are still up, so they count towards uptime. Down checks
	// are often timeouts and would drown the average, so they stay out of it.
	if total > 0 {
		row.UptimePct = new(float64(alive) / float64(total) * 100)
	}
	if alive > 0 {
		row.AvgResponseTime = new(sum / alive)
	}
	return row
}

// buildPoints walks the window on step boundaries and fills the holes the
// aggregation left behind. A bucket without checks has to show up as a bucket
// without checks, otherwise a gap in the history looks like a healthy one.
func buildPoints(m *db.Monitor, rows []*db.GetCheckWindowRow, since, now, step int64) []DataPoint {
	byTS := make(map[int64]*db.GetCheckWindowRow, len(rows))
	for _, r := range rows {
		byTS[r.Ts] = r
	}

	points := make([]DataPoint, 0, (now-since)/step+1)
	for ts := since; ts <= now; ts += step {
		point := DataPoint{Timestamp: time.Unix(ts, 0)}
		// A bucket shorter than the check interval is empty by construction, so
		// nothing can be missing from it. Push monitors only report when pinged.
		if step >= m.CheckInterval && m.Type != checker.TypePush {
			point.Expected = step / m.CheckInterval
		}

		if r, ok := byTS[ts]; ok {
			point.HasData = r.Total > 0
			point.Total = r.Total
			point.Up = r.Up
			point.Degraded = r.Degraded
			point.Down = r.Down
			if alive := r.Up + r.Degraded; alive > 0 {
				point.AvgMs = new(r.SumMs / alive)
			}
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
	if last.ResponseTime > m.DegradedThreshold {
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

// percentileAt returns the nearest-rank percentile of an ascending sample.
func percentileAt(sorted []int64, p int) int64 {
	rank := (int64(p)*int64(len(sorted)) + 99) / 100
	return sorted[rank-1]
}
