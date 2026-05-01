package api

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
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
	stats, err := s.cfg.Conn.Q.GetMonitorStats(r.Context(), &secondsStr)
	if err != nil {
		http.Error(w, "Failed to get monitor stats", http.StatusInternalServerError)
		return
	}

	since := time.Now().Add(-time.Duration(seconds) * time.Second)
	slog.Debug("GetMonitors", "seconds", seconds, "since", since)

	responseTimes, err := s.cfg.Conn.Q.GetResponseTimes(r.Context(), since)
	if err != nil {
		http.Error(w, "Failed to get response times", http.StatusInternalServerError)
		return
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

	pointsByMonitor, err := s.getDataPoints(r.Context(), seconds, since)
	if err != nil {
		http.Error(w, "Failed to get data points", http.StatusInternalServerError)
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

func (s *Server) getDataPoints(
	ctx context.Context,
	seconds int64,
	since time.Time,
) (map[int64][]DataPoint, error) {
	bucketSize := s.computeBucketSize(seconds)

	rows, err := s.cfg.Conn.Q.GetDataPoints(ctx, &db.GetDataPointsParams{
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
		if total == 0 {
			total = 1 // prevent division by zero
		}

		dp := DataPoint{
			Timestamp:    time.Unix(row.BucketTs, 0),
			ResponseTime: row.AvgResponseTime,
			IsUp:         row.UpCount > total/2,
		}

		if s.cfg.ChartType == "bars" {
			dp.UpRatio = row.UpCount / total
			dp.DegradedRatio = row.DegradedCount / total
			dp.DownRatio = row.DownCount / total
		}

		result[row.MonitorID] = append(result[row.MonitorID], dp)
	}
	return result, nil
}

func (s *Server) computeBucketSize(seconds int64) int64 {
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
