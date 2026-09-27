package config

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
)

// Sync makes the monitors table match the config and returns the result.
// Monitors are matched by name, so changing a url keeps the history.
func Sync(ctx context.Context, q *db.Queries, monitors []Monitor) ([]*db.Monitor, error) {
	existing, err := q.GetMonitors(ctx)
	if err != nil {
		return nil, err
	}
	stale := make(map[string]*db.Monitor, len(existing))
	for _, m := range existing {
		stale[m.Name] = m
	}

	for _, m := range monitors {
		want := params(m)
		old, found := stale[m.Name]
		delete(stale, m.Name)
		if found && unchanged(old, want) {
			continue
		}

		if _, err := q.UpsertMonitor(ctx, want); err != nil {
			return nil, err
		}
		if found {
			slog.Info("Updated monitor", "name", m.Name)
		} else {
			slog.Info("Added monitor", "name", m.Name)
		}
	}

	for name, m := range stale {
		if err := q.DeleteMonitor(ctx, m.ID); err != nil {
			return nil, err
		}
		slog.Info("Removed monitor", "name", name)
	}

	return q.GetMonitors(ctx)
}

func params(m Monitor) *db.UpsertMonitorParams {
	scheme := ""
	if u, err := url.Parse(m.URL); err == nil {
		scheme = u.Scheme
	}
	return &db.UpsertMonitorParams{
		Name:              m.Name,
		Url:               m.URL,
		Type:              checker.TypeForScheme(scheme),
		GroupName:         m.Group,
		CheckInterval:     m.CheckInterval,
		Retries:           m.Retries,
		DegradedThreshold: m.DegradedThreshold,
		ExpectedStatus:    m.ExpectedStatus,
		Keyword:           m.Keyword,
		IgnoreCertExpiry:  m.IgnoreCertExpiry,
	}
}

func unchanged(old *db.Monitor, want *db.UpsertMonitorParams) bool {
	return *want == db.UpsertMonitorParams{
		Name:              old.Name,
		Url:               old.Url,
		Type:              old.Type,
		GroupName:         old.GroupName,
		CheckInterval:     old.CheckInterval,
		Retries:           old.Retries,
		DegradedThreshold: old.DegradedThreshold,
		ExpectedStatus:    old.ExpectedStatus,
		Keyword:           old.Keyword,
		IgnoreCertExpiry:  old.IgnoreCertExpiry,
	}
}
