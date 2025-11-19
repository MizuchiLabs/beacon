package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mizuchilabs/beacon/internal/db"
)

// handleCreateMonitor creates a new monitor
func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		CheckInterval int64  `json:"check_interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.URL == "" {
		respondError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	if req.CheckInterval == 0 {
		req.CheckInterval = 60 // default to 60 seconds
	}

	monitor, err := s.cfg.Conn.Queries.CreateMonitor(r.Context(), &db.CreateMonitorParams{
		Name:          req.Name,
		Url:           req.URL,
		CheckInterval: req.CheckInterval,
		IsActive:      true,
	})
	if err != nil {
		slog.Error("failed to create monitor", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}

	respondJSON(w, http.StatusCreated, monitor)
}

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

// handleUpdateMonitor updates an existing monitor
func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	var req struct {
		Name          *string `json:"name"`
		URL           *string `json:"url"`
		CheckInterval *int64  `json:"check_interval"`
		IsActive      *bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params := db.UpdateMonitorParams{ID: id}
	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.URL != nil {
		params.Url = *req.URL
	}
	if req.CheckInterval != nil {
		params.CheckInterval = *req.CheckInterval
	}
	if req.IsActive != nil {
		params.IsActive = *req.IsActive
	}

	monitor, err := s.cfg.Conn.Queries.UpdateMonitor(r.Context(), &params)
	if err != nil {
		slog.Error("failed to update monitor", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}

	respondJSON(w, http.StatusOK, monitor)
}

// handleDeleteMonitor deletes a monitor
func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid monitor id")
		return
	}

	if err := s.cfg.Conn.Queries.DeleteMonitor(r.Context(), id); err != nil {
		slog.Error("failed to delete monitor", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
