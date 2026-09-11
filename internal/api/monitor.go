package api

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/db"
)

// degradedThreshold is the response time above which an up check counts as degraded.
const degradedThreshold = 500

// MonitorStats is the aggregated uptime and latency stats for one monitor.
// Uptime, latency and percentiles are null when the window holds no checks.
type MonitorStats struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	URL             string       `json:"url"`
	CheckInterval   int64        `json:"check_interval"`
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

// DataPoint is one aggregated check bucket on the chart.
type DataPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	ResponseTime  int64     `json:"response_time"`
	IsUp          bool      `json:"is_up"`
	UpRatio       float64   `json:"up_ratio"`
	DegradedRatio float64   `json:"degraded_ratio"`
	DownRatio     float64   `json:"down_ratio"`
}

type GetMonitorsInput struct {
	Seconds int64 `query:"seconds" default:"86400" minimum:"60" maximum:"31536000" doc:"How far back to aggregate, in seconds"`
}

type MonitorsOutput struct {
	Body []MonitorStats
}

type MonitorService struct {
	cfg *Config
	q   *db.Queries
}

func NewMonitorService(api huma.API, cfg *Config, q *db.Queries) *MonitorService {
	svc := &MonitorService{cfg: cfg, q: q}
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
	since := time.Now().Unix() - in.Seconds

	stats, err := s.q.GetMonitorStats(ctx, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get monitor stats")
	}

	responseTimes, err := s.q.GetResponseTimes(ctx, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get response times")
	}

	timesByMonitor := make(map[int64][]int64)
	for _, rt := range responseTimes {
		timesByMonitor[rt.MonitorID] = append(timesByMonitor[rt.MonitorID], rt.ResponseTime)
	}

	percentilesByMonitor := make(map[int64]*Percentiles)
	for monitorID, times := range timesByMonitor {
		if len(times) == 0 {
			continue
		}
		slices.Sort(times)
		percentilesByMonitor[monitorID] = &Percentiles{
			P50: percentile(times, 50),
			P75: percentile(times, 75),
			P90: percentile(times, 90),
			P95: percentile(times, 95),
			P99: percentile(times, 99),
		}
	}

	pointsByMonitor, err := s.getDataPoints(ctx, in.Seconds, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get data points")
	}

	result := make([]MonitorStats, len(stats))
	for i, stat := range stats {
		row := MonitorStats{
			ID:            stat.ID,
			Name:          stat.Name,
			URL:           stat.Url,
			CheckInterval: stat.CheckInterval,
			Datapoints:    pointsByMonitor[stat.ID],
		}
		if stat.CheckCount > 0 {
			row.UptimePct = &stat.UptimePct
			row.AvgResponseTime = &stat.AvgResponseTime
			row.Percentiles = percentilesByMonitor[stat.ID]
		}
		result[i] = row
	}

	return &MonitorsOutput{Body: result}, nil
}

// percentile returns the nearest-rank percentile of an ascending slice.
func percentile(sorted []int64, p int) int64 {
	idx := (p*len(sorted)+99)/100 - 1
	return sorted[max(idx, 0)]
}

func (s *MonitorService) getDataPoints(
	ctx context.Context,
	seconds int64,
	since int64,
) (map[int64][]DataPoint, error) {
	bucketSize := s.computeBucketSize(seconds)

	rows, err := s.q.GetDataPoints(ctx, &db.GetDataPointsParams{
		BucketSize:        bucketSize,
		DegradedThreshold: degradedThreshold,
		Since:             since,
	})
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]DataPoint)
	for _, row := range rows {
		total := float64(row.TotalCount)
		upRatio := float64(row.UpCount) / total

		result[row.MonitorID] = append(result[row.MonitorID], DataPoint{
			Timestamp:     time.Unix(row.BucketTs, 0),
			ResponseTime:  row.AvgResponseTime,
			IsUp:          upRatio > 0.5,
			UpRatio:       upRatio,
			DegradedRatio: float64(row.DegradedCount) / total,
			DownRatio:     float64(row.DownCount) / total,
		})
	}
	return result, nil
}

func (s *MonitorService) computeBucketSize(seconds int64) int64 {
	if s.cfg.ChartType == "bars" {
		size := seconds / 80
		if size == 0 {
			return 1
		}
		return size
	}
	// area chart buckets
	switch {
	case seconds <= 86400:
		return 1800
	case seconds <= 604800:
		return 14400
	case seconds <= 1209600:
		return 28800
	default:
		return 86400
	}
}
