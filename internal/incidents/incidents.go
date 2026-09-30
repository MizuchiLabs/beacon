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

	"github.com/mizuchilabs/beacon/internal/notify"
)

// syncRef is owned by beacon. origin/HEAD is a symbolic ref in git CLI clones,
// which go-git refuses to overwrite during a fetch.
const syncRef plumbing.ReferenceName = "refs/beacon/incidents"

// notifyWindow is how recent a new incident or update must be to notify
// anyone. Older ones are history being written down, not news.
const notifyWindow = time.Hour

type Service struct {
	notifier *notify.Service
	trigger  chan struct{}

	mu        sync.RWMutex
	incidents []Incident
	loaded    bool
	monitors  []string
	warned    map[string]bool

	RepoURL   string        `env:"BEACON_INCIDENT_REPO"`
	RepoPath  string        `env:"BEACON_INCIDENT_PATH"`
	Interval  time.Duration `env:"BEACON_INCIDENT_SYNC"       envDefault:"5m"`
	SyncToken string        `env:"BEACON_INCIDENT_SYNC_TOKEN"`
}

// New returns nil when there is no incident repo and no incident directory,
// which turns incidents off. notifier may be nil.
func New(dataDir string, notifier *notify.Service) (*Service, error) {
	i, err := env.ParseAs[Service]()
	if err != nil {
		return nil, err
	}
	i.notifier = notifier
	i.trigger = make(chan struct{}, 1)
	i.warned = map[string]bool{}
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
// restart. Sync asks for an extra round in between.
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
			case <-i.trigger:
				i.sync(ctx)
			}
		}
	}()
}

// Sync queues a sync right away, e.g. from a git push webhook. Calls while
// one is queued are dropped.
func (i *Service) Sync() {
	select {
	case i.trigger <- struct{}{}:
	default:
	}
}

func (i *Service) sync(ctx context.Context) {
	if i.RepoURL != "" {
		if err := i.syncRepo(ctx); err != nil {
			slog.Error("Failed to sync incidents repo", "error", err)
		}
	}
	news, err := i.loadIncidents()
	if err != nil {
		slog.Error("Failed to load incidents", "error", err)
		return
	}
	if i.notifier == nil {
		return
	}
	for _, e := range news {
		slog.Info("Notifying about incident", "incident", e.ID, "status", e.Status)
		i.notifier.Incident(ctx, e)
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

// loadIncidents swaps in the incidents from disk and returns what is new
// since the last load. The first load only fills the list.
func (i *Service) loadIncidents() ([]notify.IncidentEvent, error) {
	incidents, err := ParseIncidentsDir(i.RepoPath)
	if err != nil {
		return nil, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	var news []notify.IncidentEvent
	if i.loaded {
		news = changes(i.incidents, incidents, time.Now())
	}
	i.incidents = incidents
	i.loaded = true
	i.warnUnknownMonitors()
	return news, nil
}

func changes(before, after []Incident, now time.Time) []notify.IncidentEvent {
	known := map[string]int{}
	for _, inc := range before {
		known[inc.ID] = len(inc.Updates)
	}

	var news []notify.IncidentEvent
	for _, inc := range after {
		count, ok := known[inc.ID]
		if ok && len(inc.Updates) <= count {
			continue
		}
		e := notify.IncidentEvent{
			ID:       inc.ID,
			Title:    inc.Title,
			Status:   inc.Status,
			Message:  inc.Description,
			Monitors: inc.AffectedMonitors,
		}
		at := inc.StartedAt
		if len(inc.Updates) > 0 {
			latest := inc.Updates[len(inc.Updates)-1]
			e.Status, e.Message, at = latest.Status, latest.Message, latest.CreatedAt
		}
		// Scheduled maintenance is news when announced, however far ahead.
		if now.Sub(at) > notifyWindow && e.Status != "scheduled" {
			continue
		}
		news = append(news, e)
	}
	return news
}

// SetMonitors tells the service which monitor names exist, so incidents that
// name an unknown one get a warning in the log.
func (i *Service) SetMonitors(names []string) {
	if i == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.monitors = names
	i.warnUnknownMonitors()
}

// warnUnknownMonitors logs each unknown name once. Callers hold mu.
func (i *Service) warnUnknownMonitors() {
	if i.monitors == nil {
		return
	}
	for _, inc := range i.incidents {
		for _, name := range UnknownMonitors(inc, i.monitors) {
			key := inc.ID + "\x00" + name
			if i.warned[key] {
				continue
			}
			i.warned[key] = true
			slog.Warn("Incident names an unknown monitor", "incident", inc.ID, "monitor", name)
		}
	}
}

// UnknownMonitors returns the affected monitors that are not in names.
func UnknownMonitors(inc Incident, names []string) []string {
	var unknown []string
	for _, name := range inc.AffectedMonitors {
		if !slices.Contains(names, name) {
			unknown = append(unknown, name)
		}
	}
	return unknown
}

func (i *Service) GetIncidents() []Incident {
	i.mu.RLock()
	defer i.mu.RUnlock()

	now := time.Now()
	out := make([]Incident, len(i.incidents))
	for n, inc := range i.incidents {
		out[n] = inc.at(now)
	}
	return out
}

func (i *Service) GetIncident(id string) (*Incident, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, inc := range i.incidents {
		if inc.ID == id {
			return new(inc.at(time.Now())), true
		}
	}
	return nil, false
}

// at resolves a maintenance window once its planned end has passed, so
// nobody has to write a resolved update at 4am.
func (inc *Incident) at(now time.Time) Incident {
	out := *inc
	if out.EndsAt == nil || out.Status == "resolved" || now.Before(*out.EndsAt) {
		return out
	}
	out.Status = "resolved"
	out.ResolvedAt = out.EndsAt
	return out
}
