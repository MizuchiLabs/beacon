package api

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
)

// MonitorStats is the aggregated uptime and latency stats for one monitor.
type MonitorStats struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	URL             string      `json:"url"`
	CheckInterval   int64       `json:"check_interval"`
	AvgResponseTime int64       `json:"avg_response_time"`
	UptimePct       float64     `json:"uptime_pct"`
	Percentiles     Percentiles `json:"percentiles"`
	Datapoints      []DataPoint `json:"data_points"`
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
	UpRatio       float64   `json:"up_ratio,omitempty"`
	DegradedRatio float64   `json:"degraded_ratio,omitempty"`
	DownRatio     float64   `json:"down_ratio,omitempty"`
}

type GetMonitorsInput struct {
	Seconds int64 `query:"seconds" default:"86400" minimum:"60" maximum:"31536000" doc:"How far back to aggregate, in seconds"`
}

type MonitorsOutput struct {
	Body []MonitorStats
}

type MonitorService struct {
	q         *db.Queries
	chartType string
}

func NewMonitorService(api huma.API, cfg *config.Config) *MonitorService {
	svc := &MonitorService{q: cfg.Conn.Q, chartType: cfg.ChartType}
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

	percentilesByMonitor := make(map[int64]Percentiles)
	for monitorID, times := range timesByMonitor {
		if len(times) == 0 {
			continue
		}
		slices.Sort(times)
		n := len(times)
		percentilesByMonitor[monitorID] = Percentiles{
			P50: times[n*50/100],
			P75: times[n*75/100],
			P90: times[n*90/100],
			P95: times[n*95/100],
			P99: times[n*99/100],
		}
	}

	pointsByMonitor, err := s.getDataPoints(ctx, in.Seconds, since)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get data points")
	}

	result := make([]MonitorStats, len(stats))
	for i, stat := range stats {
		result[i] = MonitorStats{
			ID:              stat.ID,
			Name:            stat.Name,
			URL:             stat.Url,
			CheckInterval:   stat.CheckInterval,
			UptimePct:       stat.UptimePct,
			AvgResponseTime: stat.AvgResponseTime,
			Percentiles:     percentilesByMonitor[stat.ID],
			Datapoints:      pointsByMonitor[stat.ID],
		}
	}

	return &MonitorsOutput{Body: result}, nil
}

func (s *MonitorService) getDataPoints(
	ctx context.Context,
	seconds int64,
	since int64,
) (map[int64][]DataPoint, error) {
	bucketSize := s.computeBucketSize(seconds)

	rows, err := s.q.GetDataPoints(ctx, &db.GetDataPointsParams{
		BucketSize:        bucketSize,
		DegradedThreshold: 500,
		Since:             since,
	})
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]DataPoint)
	for _, row := range rows {
		total := float64(row.TotalCount)

		dp := DataPoint{
			Timestamp:    time.Unix(row.BucketTs, 0),
			ResponseTime: row.AvgResponseTime,
			IsUp:         float64(row.UpCount) > total/2,
		}

		if s.chartType == "bars" {
			dp.UpRatio = float64(row.UpCount) / total
			dp.DegradedRatio = float64(row.DegradedCount) / total
			dp.DownRatio = float64(row.DownCount) / total
		}

		result[row.MonitorID] = append(result[row.MonitorID], dp)
	}
	return result, nil
}

func (s *MonitorService) computeBucketSize(seconds int64) int64 {
	if s.chartType == "bars" {
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
