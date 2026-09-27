// Package incidents provides functionality for syncing incidents
package incidents

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
)

type Service struct {
	mu        sync.RWMutex
	incidents []Incident

	RepoURL  string        `env:"BEACON_INCIDENT_REPO"`
	RepoPath string        `env:"BEACON_INCIDENT_PATH"`
	Interval time.Duration `env:"BEACON_INCIDENT_SYNC" envDefault:"5m"`
}

// New returns nil when there is no incident repo and no incident directory,
// which turns incidents off.
func New(dataDir string) (*Service, error) {
	i, err := env.ParseAs[Service]()
	if err != nil {
		return nil, err
	}
	if i.RepoPath == "" {
		i.RepoPath = filepath.Join(dataDir, "incidents")
	}

	if i.RepoURL == "" {
		if _, err := os.Stat(i.RepoPath); os.IsNotExist(err) {
			slog.Debug("No incident repo or directory, incidents disabled", "path", i.RepoPath)
			return nil, nil
		}
		slog.Info("Using local incident directory", "path", i.RepoPath)
	}
	return &i, nil
}

// Start loads the incidents and keeps them in sync until ctx is done. A local
// directory is re-read on the same interval, so new files show up without a
// restart.
func (i *Service) Start(ctx context.Context) {
	if i == nil {
		return
	}

	i.sync(ctx)
	go func() {
		tick := time.Tick(i.Interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				i.sync(ctx)
			}
		}
	}()
}

func (i *Service) sync(ctx context.Context) {
	if i.RepoURL != "" {
		if err := i.syncRepo(ctx); err != nil {
			slog.Error("Failed to sync incidents repo", "error", err)
		}
	}
	if err := i.loadIncidents(); err != nil {
		slog.Error("Failed to load incidents", "error", err)
	}
}

// syncRepo mirrors the remote. Fetch and hard reset instead of pull, so a
// force push to the incidents repo can't wedge the sync.
func (i *Service) syncRepo(ctx context.Context) error {
	if _, err := os.Stat(i.RepoPath); os.IsNotExist(err) {
		slog.Info("Cloning incidents repository", "url", i.RepoURL)
		return run(ctx, "git", "clone", "--depth", "1", i.RepoURL, i.RepoPath)
	}

	slog.Debug("Pulling latest incidents from repository")
	if err := run(ctx, "git", "-C", i.RepoPath, "fetch", "--depth", "1", "origin"); err != nil {
		return err
	}
	return run(ctx, "git", "-C", i.RepoPath, "reset", "--hard", "FETCH_HEAD")
}

func run(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput() // #nosec G204
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, args[len(args)-1], err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (i *Service) loadIncidents() error {
	incidents, err := ParseIncidentsDir(i.RepoPath)
	if err != nil {
		return err
	}

	i.mu.Lock()
	i.incidents = incidents
	i.mu.Unlock()
	return nil
}

func (i *Service) GetIncidents() []Incident {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return slices.Clone(i.incidents)
}

func (i *Service) GetIncident(id string) (*Incident, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, incident := range i.incidents {
		if incident.ID == id {
			return &incident, true
		}
	}

	return nil, false
}
