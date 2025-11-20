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
	ServerPort   string `env:"BEACON_PORT"     envDefault:"3000"`
	DBPath       string `env:"BEACON_DB_PATH"  envDefault:"data/beacon.db"`
	Insecure     bool   `env:"BEACON_INSECURE" envDefault:"false"`
	ConfigPath   string `env:"BEACON_CONFIG"   envDefault:"config.yaml"`
	MonitorsYAML string `env:"BEACON_MONITORS"`

	// Frontend settings
	Title       string `env:"BEACON_TITLE"       envDefault:"Beacon Dashboard"`
	Description string `env:"BEACON_DESCRIPTION" envDefault:"Track uptime and response times across all monitors"`
	Timezone    string `env:"BEACON_TIMEZONE"    envDefault:"Europe/Vienna"`

	Debug bool `env:"DEBUG" envDefault:"false"`
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

	if cmd.String("port") != "" {
		cfg.ServerPort = cmd.String("port")
	}
	if cmd.String("config") != "" {
		cfg.ConfigPath = cmd.String("config")
	}

	Logger(&cfg)
	cfg.Conn = db.NewConnection(cfg.DBPath)
	cfg.Checker = checker.New(cfg.Insecure)
	cfg.Notifier = notify.New(ctx, cfg.Conn)
	cfg.Scheduler = scheduler.New(cfg.Conn, cfg.Checker, cfg.Notifier)
	cfg.Incidents = incidents.New()

	// Sync monitors to DB
	if err := cfg.syncMonitors(ctx); err != nil {
		log.Fatalf("Failed to sync monitors to DB: %v", err)
	}

	return &cfg
}

func Logger(cfg *Config) {
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
