package incidents

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
)

type IncidentManager struct {
	RepoURL   string        `env:"BEACON_INCIDENT_REPO"`
	RepoPath  string        `env:"BEACON_INCIDENT_PATH"`
	Interval  time.Duration `env:"BEACON_INCIDENT_SYNC" envDefault:"5m"`
	mu        sync.RWMutex
	incidents []Incident
}

func New() *IncidentManager {
	s, err := env.ParseAs[IncidentManager]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	if s.RepoURL == "" {
		if _, err := os.Stat(s.RepoPath); os.IsNotExist(err) {
			slog.Debug("No incident repo URL or local path set, incidents disabled")
			return nil
		}
		slog.Info("Using local incident directory", "path", s.RepoPath)
	}

	return &IncidentManager{
		RepoURL:   s.RepoURL,
		RepoPath:  s.RepoPath,
		Interval:  s.Interval,
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
		cmd := exec.Command("git", "clone", "--depth", "1", i.RepoURL, i.RepoPath)
		return cmd.Run()
	}

	slog.Debug("Pulling latest incidents from repository")
	cmd := exec.Command("git", "-C", i.RepoPath, "pull", "--rebase")
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

	slog.Info("Loaded incidents", "count", len(incidents))
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

func (s *IncidentManager) GetIncidentByID(id string) (*Incident, bool) {
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
