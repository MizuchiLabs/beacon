package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/util"
)

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

type Percentiles struct {
	P50 int64 `json:"p50"`
	P75 int64 `json:"p75"`
	P90 int64 `json:"p90"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
}

type DataPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	ResponseTime  int64     `json:"response_time"`
	IsUp          bool      `json:"is_up"`
	UpRatio       float64   `json:"up_ratio,omitempty"`
	DegradedRatio float64   `json:"degraded_ratio,omitempty"`
	DownRatio     float64   `json:"down_ratio,omitempty"`
}

func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	util.RespondJSON(w, http.StatusOK, map[string]any{
		"title":             s.cfg.Title,
		"description":       s.cfg.Description,
		"timezone":          s.cfg.Timezone,
		"chart_type":        s.cfg.ChartType,
		"incidents_enabled": s.cfg.Incidents != nil,
	})
}

func (s *Server) GetMonitors(w http.ResponseWriter, r *http.Request) {
	secondsStr := r.URL.Query().Get("seconds")
	if secondsStr == "" {
		secondsStr = "86400"
	}

	seconds, _ := strconv.ParseInt(secondsStr, 10, 64)

	stats, err := s.cfg.Conn.Queries.GetMonitorStats(r.Context(), &secondsStr)
	if err != nil {
		slog.Error("failed to get monitor stats", "error", err)
		util.RespondError(w, http.StatusInternalServerError, "failed to retrieve monitors")
		return
	}

	since := time.Now().Add(-time.Duration(seconds) * time.Second)
	slog.Debug("GetMonitors", "seconds", seconds, "since", since)

	percentiles, err := s.cfg.Conn.Queries.GetPercentiles(r.Context(), since)
	if err != nil {
		slog.Error("failed to get percentiles", "error", err)
		util.RespondError(w, http.StatusInternalServerError, "failed to retrieve percentiles")
		return
	}
	percentilesByMonitor := make(map[int64]Percentiles)
	for _, p := range percentiles {
		percentilesByMonitor[p.MonitorID] = Percentiles{
			P50: p.P50, P75: p.P75, P90: p.P90, P95: p.P95, P99: p.P99,
		}
	}

	var pointsByMonitor map[int64][]DataPoint
	if s.cfg.ChartType == "bars" {
		pointsByMonitor, err = s.getStatusDataPoints(r.Context(), seconds, since)
	} else {
		pointsByMonitor, err = s.getTimeSeriesDataPoints(r.Context(), seconds, since)
	}
	if err != nil {
		slog.Error("failed to get data points", "error", err)
		util.RespondError(w, http.StatusInternalServerError, "failed to retrieve chart data")
		return
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

	util.RespondJSON(w, http.StatusOK, result)
}

func (s *Server) getStatusDataPoints(
	ctx context.Context,
	seconds int64,
	since time.Time,
) (map[int64][]DataPoint, error) {
	bucketSize := seconds / 80
	if bucketSize == 0 {
		bucketSize = 1
	}

	params := &db.GetStatusDataPointsParams{
		BucketSize:        bucketSize,
		DegradedThreshold: 500,
		Since:             since,
	}

	rows, err := s.cfg.Conn.Queries.GetStatusDataPoints(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]DataPoint)
	for _, row := range rows {
		total := float64(row.TotalCount)
		result[row.MonitorID] = append(result[row.MonitorID], DataPoint{
			Timestamp:     time.Unix(row.BucketTs, 0),
			ResponseTime:  0,
			IsUp:          row.UpCount > float64(row.TotalCount)/2,
			UpRatio:       row.UpCount / total,
			DegradedRatio: row.DegradedCount / total,
			DownRatio:     row.DownCount / total,
		})
	}
	return result, nil
}

func (s *Server) getTimeSeriesDataPoints(
	ctx context.Context,
	seconds int64,
	since time.Time,
) (map[int64][]DataPoint, error) {
	rows, err := s.cfg.Conn.Queries.GetTimeSeriesDataPoints(ctx, &db.GetTimeSeriesDataPointsParams{
		BucketSize: computeBucketSize(seconds),
		Since:      since,
	})
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]DataPoint)
	for _, row := range rows {
		result[row.MonitorID] = append(result[row.MonitorID], DataPoint{
			Timestamp:    time.Unix(row.BucketTs, 0),
			ResponseTime: row.AvgResponseTime,
			IsUp:         row.UpCount > float64(row.TotalCount)/2,
		})
	}
	return result, nil
}

func computeBucketSize(seconds int64) int64 {
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
