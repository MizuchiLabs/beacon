// Package config loads the monitors and notification webhooks from YAML and
// syncs the monitors into the database.
package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"

	"github.com/mizuchilabs/beacon/internal/checker"
)

// watchInterval is how often the config file is checked for changes.
const watchInterval = 5 * time.Second

type Config struct {
	Monitors []Monitor `yaml:"monitors"`
	Webhooks []Webhook `yaml:"webhooks"`
}

type Monitor struct {
	Name              string `yaml:"name"`
	URL               string `yaml:"url"`
	Group             string `yaml:"group"`
	CheckInterval     int64  `yaml:"check_interval"`
	Retries           int64  `yaml:"retries"`
	DegradedThreshold int64  `yaml:"degraded_threshold"`
	ExpectedStatus    int64  `yaml:"expected_status"`
	Keyword           string `yaml:"keyword"`
	IgnoreCertExpiry  bool   `yaml:"ignore_cert_expiry"`
}

// Webhook receives every notification as a POST. Body is a Go template, the
// event is sent as JSON when it is empty.
type Webhook struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type source struct {
	InlineYAML string `env:"BEACON_MONITORS"`
}

// UnmarshalYAML fills in the defaults so only name and url are required.
func (m *Monitor) UnmarshalYAML(node *yaml.Node) error {
	type plain Monitor
	p := plain{CheckInterval: 60, Retries: 1, DegradedThreshold: 500}
	if err := node.Decode(&p); err != nil {
		return err
	}
	*m = Monitor(p)
	return nil
}

// Load reads the config from BEACON_MONITORS or, when that is empty, from the
// file at path. A missing file is an empty config.
func Load(path string) (*Config, error) {
	inline, err := inlineYAML()
	if err != nil {
		return nil, err
	}
	if inline != "" {
		slog.Debug("Loading config from BEACON_MONITORS")
		return parse([]byte(inline))
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		slog.Warn("Config file not found, no monitors configured", "path", path)
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	slog.Debug("Loading config file", "path", path)
	return parse(data)
}

// Watch calls apply with the new config whenever the file at path changes.
// Polling instead of fsnotify also catches bind mounts and swapped symlinks.
func Watch(ctx context.Context, path string, apply func(*Config) error) {
	if inline, err := inlineYAML(); err != nil || inline != "" {
		return
	}

	last, _ := os.ReadFile(path)
	tick := time.Tick(watchInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
		}

		data, err := os.ReadFile(path)
		if err != nil || bytes.Equal(data, last) {
			continue
		}
		last = data

		cfg, err := parse(data)
		if err != nil {
			slog.Error("Config changed but is invalid, keeping the old one", "error", err)
			continue
		}
		if err := apply(cfg); err != nil {
			slog.Error("Failed to apply new config", "error", err)
			continue
		}
		slog.Info("Config reloaded", "monitors", len(cfg.Monitors), "webhooks", len(cfg.Webhooks))
	}
}

func inlineYAML() (string, error) {
	src, err := env.ParseAs[source]()
	return src.InlineYAML, err
}

func parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validate(cfg *Config) error {
	names := make(map[string]bool, len(cfg.Monitors))
	urls := make(map[string]string, len(cfg.Monitors))

	for i, m := range cfg.Monitors {
		if strings.TrimSpace(m.Name) == "" {
			return fmt.Errorf("monitor #%d: name is required", i+1)
		}
		if names[m.Name] {
			return fmt.Errorf("monitor %q: duplicate name", m.Name)
		}
		names[m.Name] = true

		if err := validateMonitor(m); err != nil {
			return fmt.Errorf("monitor %q: %w", m.Name, err)
		}

		if prev, ok := urls[m.URL]; ok {
			return fmt.Errorf("monitor %q: url %q is already used by %q", m.Name, m.URL, prev)
		}
		urls[m.URL] = m.Name
	}

	for i, w := range cfg.Webhooks {
		u, err := url.Parse(w.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("webhook #%d: url must be an http or https url", i+1)
		}
	}
	return nil
}

func validateMonitor(m Monitor) error {
	u, err := url.Parse(m.URL)
	switch {
	case strings.TrimSpace(m.URL) == "":
		return errors.New("url is required")
	case err != nil:
		return fmt.Errorf("invalid url: %w", err)
	case !checker.ValidScheme(u.Scheme):
		return fmt.Errorf("url scheme must be one of %s, got %q", strings.Join(checker.Schemes, ", "), u.Scheme)
	case u.Host == "":
		return errors.New("url must have a host")
	case u.Scheme == "tcp" && u.Port() == "":
		return errors.New("tcp url must include a port, e.g. tcp://db.example.com:5432")
	case m.CheckInterval < 10 || m.CheckInterval > 86400:
		return fmt.Errorf("check_interval must be between 10 and 86400 seconds, got %d", m.CheckInterval)
	case m.Retries < 0 || m.Retries > 10:
		return fmt.Errorf("retries must be between 0 and 10, got %d", m.Retries)
	case m.DegradedThreshold < 1:
		return fmt.Errorf("degraded_threshold must be a positive number of ms, got %d", m.DegradedThreshold)
	case m.ExpectedStatus != 0 && (m.ExpectedStatus < 100 || m.ExpectedStatus > 599):
		return fmt.Errorf("expected_status must be a valid HTTP status, got %d", m.ExpectedStatus)
	}

	isHTTP := checker.TypeForScheme(u.Scheme) == checker.TypeHTTP
	if !isHTTP && (m.ExpectedStatus != 0 || m.Keyword != "") {
		return errors.New("expected_status and keyword only work with http and https urls")
	}
	return nil
}
