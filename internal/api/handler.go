package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mizuchilabs/beacon/internal/db"
)

// handleGetMonitor retrieves a single monitor by ID
func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
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
func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.cfg.Conn.Queries.GetMonitors(r.Context())
	if err != nil {
		slog.Error("failed to list monitors", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve monitors")
		return
	}

	respondJSON(w, http.StatusOK, monitors)
}

// handleGetMonitorStatus retrieves monitor with its latest check status
func (s *Server) handleGetMonitorStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	status, err := s.cfg.Conn.Queries.GetMonitorStatus(r.Context(), id)
	if err != nil {
		slog.Error("failed to get monitor status", "error", err)
		respondError(w, http.StatusNotFound, "monitor not found")
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// handleGetUptimeStats retrieves uptime statistics for a monitor (last 24h)
func (s *Server) handleGetUptimeStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	stats, err := s.cfg.Conn.Queries.GetUptimeStats(r.Context(), id)
	if err != nil {
		slog.Error("failed to get uptime stats", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve stats")
		return
	}

	// Calculate uptime percentage
	uptimePercent := 0.0
	if stats.TotalChecks > 0 {
		uptimePercent = (float64(*stats.SuccessfulChecks) / float64(stats.TotalChecks)) * 100
	}

	response := map[string]any{
		"total_checks":      stats.TotalChecks,
		"successful_checks": stats.SuccessfulChecks,
		"avg_response_time": stats.AvgResponseTime,
		"uptime_percentage": uptimePercent,
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetMonitorStats retrieves time-series stats for a monitor
func (s *Server) handleGetMonitorStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	// Get time range from query param (default: 7 days)
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "-7 days"
	}

	// Validate time range
	validRanges := map[string]bool{
		"-7 days":  true,
		"-14 days": true,
		"-30 days": true,
	}
	if !validRanges[timeRange] {
		timeRange = "-7 days"
	}

	stats, err := s.cfg.Conn.Queries.GetCheckStats(r.Context(), &db.GetCheckStatsParams{
		MonitorID: id,
		Datetime:  timeRange,
	})
	if err != nil {
		slog.Error("failed to get monitor stats", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve stats")
		return
	}

	// Transform to chart-friendly format
	chartData := make([]map[string]any, len(stats))
	for i, stat := range stats {
		uptimePercent := 0.0
		if stat.TotalChecks > 0 {
			uptimePercent = (float64(stat.SuccessfulChecks) / float64(stat.TotalChecks)) * 100
		}

		chartData[i] = map[string]any{
			"timestamp":         stat.HourTimestamp,
			"uptime_percent":    uptimePercent,
			"response_time":     stat.AvgResponseTime,
			"total_checks":      stat.TotalChecks,
			"successful_checks": stat.SuccessfulChecks,
		}
	}

	respondJSON(w, http.StatusOK, chartData)
}

// handleGetCheckHistory retrieves check history for a monitor
func (s *Server) handleGetCheckHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	limit := int64(100)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.ParseInt(limitParam, 10, 64); err == nil {
			limit = parsedLimit
		}
	}

	checks, err := s.cfg.Conn.Queries.GetChecks(r.Context(), &db.GetChecksParams{
		MonitorID: id,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("failed to get check history", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve checks")
		return
	}

	respondJSON(w, http.StatusOK, checks)
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
