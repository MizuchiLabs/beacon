package scheduler

import (
	"context"
	"sync"

	"github.com/mizuchilabs/beacon/internal/db"
)

type incidentTracker struct {
	conn            *db.Connection
	activeIncidents map[int64]int64 // monitor_id -> incident_id
	mu              sync.Mutex
}

func newIncidentTracker(conn *db.Connection) *incidentTracker {
	return &incidentTracker{
		conn:            conn,
		activeIncidents: make(map[int64]int64),
	}
}

func (i *incidentTracker) Track(ctx context.Context, monitorID int64, isUp bool, reason string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	incidentID, hasActiveIncident := i.activeIncidents[monitorID]

	if !isUp && !hasActiveIncident {
		// Start new incident
		params := &db.CreateIncidentParams{
			MonitorID: monitorID,
		}
		if reason != "" {
			params.Reason = &reason
		}
		incident, err := i.conn.Queries.CreateIncident(ctx, params)
		if err == nil {
			i.activeIncidents[monitorID] = incident.ID
		}
	} else if isUp && hasActiveIncident {
		// Resolve incident
		_, err := i.conn.Queries.ResolveIncident(ctx, incidentID)
		if err == nil {
			delete(i.activeIncidents, monitorID)
		}
	}
}
