package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mizuchilabs/beacon/internal/db"
)

type MonitorStats struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	URL             string      `json:"url"`
	CheckInterval   int64       `json:"check_interval"`
	UptimePct       float64     `json:"uptime_pct"`
	AvgResponseTime *int64      `json:"avg_response_time"`
	Percentiles     Percentiles `json:"percentiles"`
	DataPoints      []DataPoint `json:"data_points"`
}

type Percentiles struct {
	P50 *int64 `json:"p50"`
	P75 *int64 `json:"p75"`
	P90 *int64 `json:"p90"`
	P95 *int64 `json:"p95"`
	P99 *int64 `json:"p99"`
}

type DataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	ResponseTime *int64    `json:"response_time"` // avg response time
	IsUp         bool      `json:"is_up"`
}

// handleGetMonitor retrieves a single monitor by ID
func (s *Server) GetMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	monitor, err := s.cfg.Conn.Queries.GetMonitor(r.Context(), id)
	if err != nil {
		slog.Error("failed to get monitor", "error", err)
		respondError(w, http.StatusNotFound, "monitor not found")
		return
	}

	respondJSON(w, http.StatusOK, monitor)
}

// handleListMonitors retrieves all active monitors
func (s *Server) ListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.cfg.Conn.Queries.GetMonitors(r.Context())
	if err != nil {
		slog.Error("failed to list monitors", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve monitors")
		return
	}

	respondJSON(w, http.StatusOK, monitors)
}

func (s *Server) GetMonitorStats(w http.ResponseWriter, r *http.Request) {
	// Get seconds from query param, default to 24 hours
	seconds := r.URL.Query().Get("seconds")
	if seconds == "" {
		seconds = "86400"
	}

	monitors, err := s.cfg.Conn.Queries.GetMonitors(r.Context())
	if err != nil {
		slog.Error("failed to get monitors", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve monitors")
		return
	}

	// Parse the JSON data_points for each monitor

	result := make([]MonitorStats, len(monitors))
	for i, monitor := range monitors {
		checks, err := s.cfg.Conn.Queries.GetChecks(r.Context(), &db.GetChecksParams{
			MonitorID: monitor.ID,
			Seconds:   &seconds,
		})
		if err != nil {
			slog.Error("failed to get checks", "error", err)
			continue
		}

		upChecks := 0
		uptimePct := 100.00

		dataPoints := aggregateDataPoints(checks, seconds)
		for _, check := range checks {
			if check.IsUp {
				upChecks++
			}
		}
		if len(checks) > 0 {
			uptimePct = (float64(upChecks) / float64(len(checks))) * 100
			uptimePct = float64(int(uptimePct*100)) / 100 // Round to 2 decimal places
		}
		result[i] = MonitorStats{
			ID:              monitor.ID,
			Name:            monitor.Name,
			URL:             monitor.Url,
			CheckInterval:   monitor.CheckInterval,
			UptimePct:       uptimePct,
			AvgResponseTime: calculateAvgResponseTime(checks),
			Percentiles:     calculatePercentiles(checks),
			DataPoints:      dataPoints,
		}
	}

	respondJSON(w, http.StatusOK, result)
}

func aggregateDataPoints(checks []*db.Check, secondsStr string) []DataPoint {
	if len(checks) == 0 {
		return []DataPoint{}
	}

	seconds, _ := strconv.ParseInt(secondsStr, 10, 64)

	// Determine bucket size based on time range
	var bucketSize time.Duration
	switch {
	case seconds <= 86400: // 24h - 30 minute buckets
		bucketSize = 30 * time.Minute
	case seconds <= 604800: // 7d - 4 hour buckets
		bucketSize = 4 * time.Hour
	case seconds <= 1209600: // 14d - 8 hour buckets
		bucketSize = 8 * time.Hour
	default: // 30d - 1 day buckets
		bucketSize = 24 * time.Hour
	}

	// Group checks into buckets
	buckets := make(map[int64][]*db.Check)
	for _, check := range checks {
		bucketKey := check.CheckedAt.Unix() / int64(bucketSize.Seconds())
		buckets[bucketKey] = append(buckets[bucketKey], check)
	}

	// Calculate average for each bucket
	dataPoints := make([]DataPoint, 0, len(buckets))
	for bucketKey, bucketChecks := range buckets {
		var sum int64
		var count int64
		upCount := 0

		for _, check := range bucketChecks {
			if check.IsUp {
				upCount++
			}
			if check.ResponseTime != nil {
				sum += *check.ResponseTime
				count++
			}
		}

		var avgResponseTime *int64
		if count > 0 {
			avg := sum / count
			avgResponseTime = &avg
		}

		dataPoints = append(dataPoints, DataPoint{
			Timestamp:    time.Unix(bucketKey*int64(bucketSize.Seconds()), 0),
			ResponseTime: avgResponseTime,
			IsUp:         upCount > len(bucketChecks)/2, // Majority up = up
		})
	}

	// Sort by timestamp
	slices.SortFunc(dataPoints, func(a, b DataPoint) int {
		return int(a.Timestamp.Unix() - b.Timestamp.Unix())
	})

	return dataPoints
}

func calculateAvgResponseTime(checks []*db.Check) *int64 {
	var sum int64
	var count int64

	for _, check := range checks {
		if check.IsUp && check.ResponseTime != nil {
			sum += *check.ResponseTime
			count++
		}
	}

	if count == 0 {
		return nil
	}

	avg := sum / count
	return &avg
}

func calculatePercentiles(checks []*db.Check) Percentiles {
	// Collect non-nil response times from successful checks
	var responseTimes []int64
	for _, check := range checks {
		if check.IsUp && check.ResponseTime != nil {
			responseTimes = append(responseTimes, *check.ResponseTime)
		}
	}

	if len(responseTimes) == 0 {
		return Percentiles{}
	}

	// Sort response times
	slices.Sort(responseTimes)

	getPercentile := func(p float64) *int64 {
		index := int(p * float64(len(responseTimes)))
		if index >= len(responseTimes) {
			index = len(responseTimes) - 1
		}
		value := responseTimes[index]
		return &value
	}

	return Percentiles{
		P50: getPercentile(0.50),
		P75: getPercentile(0.75),
		P90: getPercentile(0.90),
		P95: getPercentile(0.95),
		P99: getPercentile(0.99),
	}
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
