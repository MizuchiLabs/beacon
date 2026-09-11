// Package incidents provides functionality for syncing incidents
package incidents

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
)

type IncidentManager struct {
	mu        sync.RWMutex
	incidents []Incident

	RepoURL  string        `env:"BEACON_INCIDENT_REPO"`
	RepoPath string        `env:"BEACON_INCIDENT_PATH"`
	Interval time.Duration `env:"BEACON_INCIDENT_SYNC" envDefault:"5m"`
}

func New() (*IncidentManager, error) {
	inc, err := env.ParseAs[IncidentManager]()
	if err != nil {
		return nil, err
	}

	if inc.RepoURL == "" {
		if _, err := os.Stat(inc.RepoPath); os.IsNotExist(err) {
			slog.Debug("No incident repo URL or local path set, incidents disabled")
			return nil, nil
		}
		slog.Info("Using local incident directory", "path", inc.RepoPath)
	}
	inc.incidents = make([]Incident, 0)

	return &inc, nil
}

func (i *IncidentManager) Start(ctx context.Context) {
	if i == nil {
		return
	}

	// Initial sync if using git
	if i.RepoURL != "" {
		if err := i.syncRepo(ctx); err != nil {
			slog.Warn("Failed initial sync, will retry...", "error", err)
		}
	}

	// Initial parse
	if err := i.loadIncidents(); err != nil {
		slog.Warn("Failed to load incidents", "error", err)
	}

	// Periodic sync only if using git
	if i.RepoURL != "" {
		ticker := time.NewTicker(i.Interval)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := i.syncRepo(ctx); err != nil {
						slog.Error("Failed to sync incidents repo", "error", err)
						continue
					}
					if err := i.loadIncidents(); err != nil {
						slog.Error("Failed to reload incidents", "error", err)
					}
				}
			}
		}()
	}
}

func (i *IncidentManager) syncRepo(ctx context.Context) error {
	if _, err := os.Stat(i.RepoPath); os.IsNotExist(err) {
		slog.Info("Cloning incidents repository", "url", i.RepoURL)
		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", i.RepoURL, i.RepoPath) // #nosec G204
		return cmd.Run()
	}

	slog.Debug("Pulling latest incidents from repository")
	cmd := exec.CommandContext(ctx, "git", "-C", i.RepoPath, "pull", "--rebase") // #nosec G204
	return cmd.Run()
}

func (i *IncidentManager) loadIncidents() error {
	incidents, err := ParseIncidentsDir(i.RepoPath)
	if err != nil {
		return err
	}

	i.mu.Lock()
	i.incidents = incidents
	i.mu.Unlock()
	return nil
}

func (i *IncidentManager) GetIncidents() []Incident {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Return a copy to prevent external modifications
	incidents := make([]Incident, len(i.incidents))
	copy(incidents, i.incidents)
	return incidents
}

func (i *IncidentManager) GetIncident(id string) (*Incident, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, incident := range i.incidents {
		if incident.ID == id {
			return &incident, true
		}
	}

	return nil, false
}
