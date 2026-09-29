// Package incidents provides functionality for syncing incidents
package incidents

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// syncRef is owned by beacon. origin/HEAD is a symbolic ref in git CLI clones,
// which go-git refuses to overwrite during a fetch.
const syncRef plumbing.ReferenceName = "refs/beacon/incidents"

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
// force push to the incidents repo can't wedge the sync. Credentials go in the
// URL (https://user:token@host/repo), the image has no git or ssh binary.
func (i *Service) syncRepo(ctx context.Context) error {
	repo, err := git.PlainOpen(i.RepoPath)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		slog.Info("Cloning incidents repository", "url", redactURL(i.RepoURL))
		_, err = git.PlainCloneContext(ctx, i.RepoPath, false, &git.CloneOptions{
			URL:          i.RepoURL,
			Depth:        1,
			SingleBranch: true,
		})
		if err != nil {
			return fmt.Errorf("cloning incidents repo: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening incidents repo: %w", err)
	}

	slog.Debug("Pulling latest incidents from repository")
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RefSpecs: []config.RefSpec{config.RefSpec("+HEAD:" + syncRef)},
		Depth:    1,
		Force:    true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetching incidents repo: %w", err)
	}

	remote, err := repo.Reference(syncRef, true)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", syncRef, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("opening incidents worktree: %w", err)
	}
	if err := wt.Reset(&git.ResetOptions{Commit: remote.Hash(), Mode: git.HardReset}); err != nil {
		return fmt.Errorf("resetting incidents repo: %w", err)
	}
	return nil
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparsable url>"
	}
	return u.Redacted()
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
