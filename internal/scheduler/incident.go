package scheduler

import (
	"context"
	"sync"

	"github.com/mizuchilabs/beacon/internal/db"
)

type incidentTracker struct {
	db              *db.Queries
	activeIncidents map[int64]int64 // monitor_id -> incident_id
	mu              sync.Mutex
}

func newIncidentTracker(db *db.Queries) *incidentTracker {
	return &incidentTracker{
		db:              db,
		activeIncidents: make(map[int64]int64),
	}
}

func (it *incidentTracker) Track(ctx context.Context, monitorID int64, isUp bool, reason string) {
	it.mu.Lock()
	defer it.mu.Unlock()

	incidentID, hasActiveIncident := it.activeIncidents[monitorID]

	if !isUp && !hasActiveIncident {
		// Start new incident
		params := &db.CreateIncidentParams{
			MonitorID: monitorID,
		}
		if reason != "" {
			params.Reason = &reason
		}
		incident, err := it.db.CreateIncident(ctx, params)
		if err == nil {
			it.activeIncidents[monitorID] = incident.ID
		}
	} else if isUp && hasActiveIncident {
		// Resolve incident
		_, err := it.db.ResolveIncident(ctx, incidentID)
		if err == nil {
			delete(it.activeIncidents, monitorID)
		}
	}
}
