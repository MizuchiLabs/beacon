// Package config provides configuration for the application.
package config

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
	"github.com/mizuchilabs/beacon/internal/notify"
	"github.com/mizuchilabs/beacon/internal/scheduler"
	"github.com/urfave/cli/v3"
)

type EnvConfig struct {
	Debug        bool   `env:"DEBUG"           envDefault:"false"`
	ServerPort   string `env:"BEACON_PORT"     envDefault:"3000"`
	DBPath       string `env:"BEACON_DB_PATH"  envDefault:"data/beacon.db"`
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
	// Environment variables
	EnvConfig

	// Application settings
	Conn      *db.Connection
	Checker   *checker.Checker
	Scheduler *scheduler.Scheduler
	Notifier  *notify.Notifier
	Incidents *incidents.IncidentManager
}

// New loads configuration from environment variables
func New(ctx context.Context, cmd *cli.Command) *Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	if cmd != nil {
		cfg.Debug = cmd.Bool("debug")
		cfg.ServerPort = cmd.String("port")
		cfg.ConfigPath = cmd.String("config")
		cfg.ChartType = cmd.String("chart-type")
	}

	if cfg.ChartType != "bars" && cfg.ChartType != "area" {
		log.Fatalf("Invalid chart type: %s", cfg.ChartType)
	}

	cfg.initLogger()
	cfg.Conn = db.NewConnection(cfg.DBPath)
	cfg.Checker = checker.New(cfg.Timeout, cfg.Insecure)
	cfg.Notifier = notify.New(ctx, cfg.Conn)
	cfg.Scheduler = scheduler.New(cfg.Conn, cfg.Checker, cfg.Notifier, cfg.RetentionDays)
	cfg.Incidents = incidents.New(cfg.RepoURL, cfg.RepoPath, cfg.Interval)

	// Sync monitors to DB
	if err := cfg.syncMonitors(ctx); err != nil {
		log.Fatalf("Failed to sync monitors to DB: %v", err)
	}

	return &cfg
}

func (cfg *Config) initLogger() {
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(colorable.NewColorable(os.Stderr), &tint.Options{
			Level:      level,
			TimeFormat: time.RFC3339,
			NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		}),
	))
}
