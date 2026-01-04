// Package incidents provides functionality for syncing incidents
package incidents

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

type IncidentManager struct {
	RepoURL   string
	RepoPath  string
	Interval  time.Duration
	mu        sync.RWMutex
	incidents []Incident
}

func New(repoURL, repoPath string, interval time.Duration) *IncidentManager {
	if repoURL == "" {
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			slog.Debug("No incident repo URL or local path set, incidents disabled")
			return nil
		}
		slog.Info("Using local incident directory", "path", repoPath)
	}

	return &IncidentManager{
		RepoURL:   repoURL,
		RepoPath:  repoPath,
		Interval:  interval,
		incidents: make([]Incident, 0),
	}
}

func (i *IncidentManager) Start(ctx context.Context) error {
	if i == nil {
		return nil
	}

	// Initial sync if using git
	if i.RepoURL != "" {
		if err := i.syncRepo(); err != nil {
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
					if err := i.syncRepo(); err != nil {
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

	return nil
}

func (i *IncidentManager) syncRepo() error {
	if _, err := os.Stat(i.RepoPath); os.IsNotExist(err) {
		slog.Info("Cloning incidents repository", "url", i.RepoURL)
		cmd := exec.Command("git", "clone", "--depth", "1", i.RepoURL, i.RepoPath) // #nosec G204
		return cmd.Run()
	}

	slog.Debug("Pulling latest incidents from repository")
	cmd := exec.Command("git", "-C", i.RepoPath, "pull", "--rebase") // #nosec G204
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

func (s *IncidentManager) GetIncident(id string) (*Incident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.incidents {
		if s.incidents[i].ID == id {
			// Return a copy
			incident := s.incidents[i]
			return &incident, true
		}
	}

	return nil, false
}
