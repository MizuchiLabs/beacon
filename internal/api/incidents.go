package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mizuchilabs/beacon/internal/util"
)

func (s *Server) GetIncidents(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incidents == nil {
		http.Error(w, "Incidents not configured", http.StatusNotFound)
		return
	}

	incidents := s.cfg.Incidents.GetIncidents()
	util.RespondJSON(w, http.StatusOK, incidents)
}

func (s *Server) GetIncident(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incidents == nil {
		http.Error(w, "Incidents not configured", http.StatusNotFound)
		return
	}

	id := chi.URLParam(r, "id")
	incident, found := s.cfg.Incidents.GetIncidentByID(id)
	if !found {
		http.Error(w, "Incident not found", http.StatusNotFound)
		return
	}

	util.RespondJSON(w, http.StatusOK, incident)
}
