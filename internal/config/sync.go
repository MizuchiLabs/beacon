package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/mizuchilabs/beacon/internal/db"
	"gopkg.in/yaml.v3"
)

type MonitorConfig struct {
	Name          string `yaml:"name"`
	URL           string `yaml:"url"`
	CheckInterval int64  `yaml:"check_interval"`
}

type MonitorsFile struct {
	Monitors []MonitorConfig `yaml:"monitors"`
}

func (cfg *Config) loadMonitors() ([]MonitorConfig, error) {
	// Priority 1: Inline YAML from environment
	if cfg.MonitorsYAML != "" {
		slog.Debug("Loading monitors from environment...")
		monitors, err := parseMonitorsYAML([]byte(cfg.MonitorsYAML))
		if err != nil {
			return nil, fmt.Errorf("failed to parse BEACON_MONITORS: %w", err)
		}
		return monitors, validateMonitors(monitors)
	}

	// Priority 2: File path
	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Config file not found, using empty monitors", "path", cfg.ConfigPath)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	slog.Debug("Loading monitors from config file", "path", cfg.ConfigPath)
	monitors, err := parseMonitorsYAML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return monitors, validateMonitors(monitors)
}

func parseMonitorsYAML(data []byte) ([]MonitorConfig, error) {
	var configFile MonitorsFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, err
	}
	return configFile.Monitors, nil
}

func validateMonitors(monitors []MonitorConfig) error {
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

func (cfg *Config) syncMonitors(ctx context.Context) error {
	monitors, err := cfg.loadMonitors()
	if err != nil {
		return err
	}

	dbMonitors, err := cfg.Conn.Q.GetMonitors(ctx)
	if err != nil {
		return err
	}

	// Build maps for O(1) lookups
	configMap := make(map[string]MonitorConfig, len(monitors))
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
				_, err := cfg.Conn.Q.UpdateMonitor(ctx, &db.UpdateMonitorParams{
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
			_, err := cfg.Conn.Q.CreateMonitor(ctx, &db.CreateMonitorParams{
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
		if err := cfg.Conn.Q.DeleteMonitor(ctx, dbMonitor.ID); err != nil {
			return err
		}
		slog.Info("Removed monitor", "url", url)
	}

	return nil
}
