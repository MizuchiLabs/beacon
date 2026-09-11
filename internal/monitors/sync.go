// Package monitors loads monitor definitions and syncs them into the database.
package monitors

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"

	"github.com/mizuchilabs/beacon/internal/db"
)

// monitor is one entry from the monitors configuration.
type monitor struct {
	Name          string `yaml:"name"`
	URL           string `yaml:"url"`
	CheckInterval int64  `yaml:"check_interval"`
}

type monitorsFile struct {
	Monitors []monitor `yaml:"monitors"`
}

// source holds where the monitor definitions come from.
type source struct {
	InlineYAML string `env:"BEACON_MONITORS"`
}

// Sync loads monitors from BEACON_MONITORS or the given file and syncs them
// into the database, creating, updating, and removing monitors as needed.
func Sync(ctx context.Context, q *db.Queries, path string) error {
	monitors, err := load(path)
	if err != nil {
		return err
	}

	dbMonitors, err := q.GetMonitors(ctx)
	if err != nil {
		return err
	}

	// Build maps for O(1) lookups
	configMap := make(map[string]monitor, len(monitors))
	for _, m := range monitors {
		configMap[m.URL] = m
	}

	dbMap := make(map[string]*db.Monitor, len(dbMonitors))
	for _, m := range dbMonitors {
		dbMap[m.Url] = m
	}

	// Upsert monitors from config
	for url, configMonitor := range configMap {
		if dbMonitor, exists := dbMap[url]; exists {
			// Only update if something changed
			if dbMonitor.Name != configMonitor.Name ||
				dbMonitor.CheckInterval != configMonitor.CheckInterval {
				_, err := q.UpdateMonitor(ctx, &db.UpdateMonitorParams{
					ID:            dbMonitor.ID,
					Name:          configMonitor.Name,
					Url:           configMonitor.URL,
					CheckInterval: configMonitor.CheckInterval,
				})
				if err != nil {
					return err
				}
				slog.Info("Updated monitor", "url", url)
			}
			delete(dbMap, url) // Remove from deletion list
		} else {
			_, err := q.CreateMonitor(ctx, &db.CreateMonitorParams{
				Name:          configMonitor.Name,
				Url:           configMonitor.URL,
				CheckInterval: configMonitor.CheckInterval,
			})
			if err != nil {
				return err
			}
			slog.Info("Added monitor", "url", url)
		}
	}

	// Delete monitors not in config
	for url, dbMonitor := range dbMap {
		if err := q.DeleteMonitor(ctx, dbMonitor.ID); err != nil {
			return err
		}
		slog.Info("Removed monitor", "url", url)
	}

	return nil
}

func load(path string) ([]monitor, error) {
	src, err := env.ParseAs[source]()
	if err != nil {
		return nil, err
	}

	// Inline YAML from the environment
	if src.InlineYAML != "" {
		slog.Debug("Loading monitors from environment...")
		monitors, err := parseYAML([]byte(src.InlineYAML))
		if err != nil {
			return nil, fmt.Errorf("failed to parse BEACON_MONITORS: %w", err)
		}
		return monitors, validate(monitors)
	}

	// File path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Config file not found, using empty monitors", "path", path)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	slog.Debug("Loading monitors from config file", "path", path)
	monitors, err := parseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return monitors, validate(monitors)
}

func parseYAML(data []byte) ([]monitor, error) {
	var file monitorsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Monitors, nil
}

func validate(monitors []monitor) error {
	if len(monitors) == 0 {
		return nil
	}

	seenURLs := make(map[string]string, len(monitors))
	seenNames := make(map[string]string, len(monitors))

	for i, m := range monitors {
		// Validate name
		if strings.TrimSpace(m.Name) == "" {
			return fmt.Errorf("monitor #%d: name is required", i+1)
		}
		if prev, exists := seenNames[m.Name]; exists {
			return fmt.Errorf("monitor #%d: duplicate name %q (also used by %q)", i+1, m.Name, prev)
		}
		seenNames[m.Name] = m.URL

		// Validate URL
		if strings.TrimSpace(m.URL) == "" {
			return fmt.Errorf("monitor %q: url is required", m.Name)
		}

		parsedURL, err := url.Parse(m.URL)
		if err != nil {
			return fmt.Errorf("monitor %q: invalid url %q: %w", m.Name, m.URL, err)
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf(
				"monitor %q: url must use http or https scheme, got %q",
				m.Name,
				parsedURL.Scheme,
			)
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("monitor %q: url must have a host", m.Name)
		}

		if prev, exists := seenURLs[m.URL]; exists {
			return fmt.Errorf("monitor %q: duplicate url %q (also used by %q)", m.Name, m.URL, prev)
		}
		seenURLs[m.URL] = m.Name

		// Validate check interval
		if m.CheckInterval < 10 {
			return fmt.Errorf(
				"monitor %q: check_interval must be at least 10 seconds, got %d",
				m.Name,
				m.CheckInterval,
			)
		}
		if m.CheckInterval > 86400 {
			return fmt.Errorf(
				"monitor %q: check_interval must be at most 86400 seconds (24h), got %d",
				m.Name,
				m.CheckInterval,
			)
		}
	}

	return nil
}
