// Package config provides configuration for the application.
package config

import (
	"context"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
	"github.com/mizuchilabs/beacon/internal/notify"
	"github.com/mizuchilabs/beacon/internal/scheduler"
	"github.com/urfave/cli/v3"
)

type EnvConfig struct {
	Debug        bool   `env:"BEACON_DEBUG"    envDefault:"false"`
	ServerPort   string `env:"BEACON_PORT"     envDefault:"3000"`
	Insecure     bool   `env:"BEACON_INSECURE" envDefault:"false"`
	ConfigPath   string `env:"BEACON_CONFIG"   envDefault:"config.yaml"`
	MonitorsYAML string `env:"BEACON_MONITORS"`

	// Frontend settings
	Title       string `env:"BEACON_TITLE"       envDefault:"Beacon Dashboard"`
	Description string `env:"BEACON_DESCRIPTION" envDefault:"Track uptime and response times across all monitors"`
	Timezone    string `env:"BEACON_TIMEZONE"    envDefault:"Europe/Vienna"`
	ChartType   string `env:"BEACON_CHART_TYPE"  envDefault:"area"` // bars or area

	// Monitor settings
	Timeout       time.Duration `env:"BEACON_TIMEOUT"        envDefault:"30s"`
	RetentionDays int           `env:"BEACON_RETENTION_DAYS" envDefault:"30"`

	// Incident settings
	RepoURL  string        `env:"BEACON_INCIDENT_REPO"`
	RepoPath string        `env:"BEACON_INCIDENT_PATH"`
	Interval time.Duration `env:"BEACON_INCIDENT_SYNC" envDefault:"5m"`
}

type Config struct {
	EnvConfig

	// Application settings
	Q         *db.Queries
	Checker   *checker.Checker
	Scheduler *scheduler.Scheduler
	Notifier  *notify.Notifier
	Incidents *incidents.IncidentManager
}

// New loads configuration from environment variables
func New(ctx context.Context, cmd *cli.Command) (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	if cmd != nil {
		cfg.Debug = cmd.Bool("debug")
		cfg.ServerPort = cmd.String("port")
		cfg.ConfigPath = cmd.String("config")
		cfg.ChartType = cmd.String("chart-type")
	}

	if cfg.ChartType != "bars" && cfg.ChartType != "area" {
		return nil, fmt.Errorf("invalid chart type: %s", cfg.ChartType)
	}

	cfg.Q = db.NewConnection(ctx)
	cfg.Checker = checker.New(cfg.Timeout, cfg.Insecure)
	cfg.Notifier, err = notify.New(ctx, cfg.Q)
	if err != nil {
		return nil, err
	}

	// Sync monitors to DB before starting background jobs
	if err := SyncMonitors(ctx, cfg.Q, cfg.EnvConfig); err != nil {
		return nil, err
	}

	cfg.Scheduler = scheduler.New(cfg.Q, cfg.Checker, cfg.Notifier, cfg.RetentionDays)
	cfg.Scheduler.Start(ctx)
	cfg.Incidents = incidents.New(cfg.RepoURL, cfg.RepoPath, cfg.Interval)
	cfg.Incidents.Start(ctx)

	return &cfg, nil
}
