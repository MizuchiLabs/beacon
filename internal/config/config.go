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
	"github.com/mizuchilabs/beacon/internal/scheduler"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type EnvConfig struct {
	// Server
	ServerHost string `env:"BEACON_HOST" envDefault:"0.0.0.0"`
	ServerPort string `env:"BEACON_PORT" envDefault:"3000"`

	// Database
	DBPath string `env:"BEACON_DB_PATH" envDefault:"data/beacon.db"`

	// Checker
	Timeout       time.Duration `env:"BEACON_TIMEOUT"        envDefault:"30s"`
	RetentionDays int           `env:"BEACON_RETENTION_DAYS" envDefault:"30"`
	Insecure      bool          `env:"BEACON_INSECURE"       envDefault:"false"`

	// Config
	ConfigPath string `env:"BEACON_CONFIG" envDefault:"config.yaml"`

	// Debug
	Debug bool `env:"DEBUG" envDefault:"false"`
}

type MonitorConfig struct {
	Name          string `yaml:"name"`
	URL           string `yaml:"url"`
	CheckInterval int64  `yaml:"check_interval"`
}

type ConfigFile struct {
	Monitors []MonitorConfig `yaml:"monitors"`
}

type Config struct {
	// Environment variables
	EnvConfig

	// Application settings
	Conn      *db.Connection
	Checker   *checker.Checker
	Scheduler *scheduler.Scheduler

	// Monitors from config
	Monitors []MonitorConfig
}

// New loads configuration from environment variables
func New(ctx context.Context, cmd *cli.Command) *Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	Logger(&cfg)
	cfg.Conn = db.NewConnection(cfg.DBPath)
	cfg.Checker = checker.New(cfg.Timeout, cfg.Insecure)
	cfg.Scheduler = scheduler.New(cfg.Conn, cfg.Checker, cfg.RetentionDays)

	// Load monitors from config file
	if err := cfg.loadMonitorsConfig(); err != nil {
		log.Fatalf("Failed to load monitors config: %v", err)
	}

	// Sync monitors to DB
	if err := cfg.syncMonitorsToDB(ctx); err != nil {
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

func (cfg *Config) loadMonitorsConfig() error {
	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Config file not found, using empty monitors", "path", cfg.ConfigPath)
			cfg.Monitors = []MonitorConfig{}
			return nil
		}
		return err
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return err
	}

	cfg.Monitors = configFile.Monitors
	return nil
}

func (cfg *Config) syncMonitorsToDB(ctx context.Context) error {
	// Get current monitors from DB
	dbMonitors, err := cfg.Conn.Queries.GetMonitors(ctx)
	if err != nil {
		return err
	}

	// Create maps for easy lookup
	configMap := make(map[string]MonitorConfig)
	for _, m := range cfg.Monitors {
		configMap[m.URL] = m
	}

	dbMap := make(map[string]*db.Monitor)
	for _, m := range dbMonitors {
		dbMap[m.Url] = m
	}

	// Add new monitors from config or update existing
	for url, configMonitor := range configMap {
		if dbMonitor, exists := dbMap[url]; exists {
			// Update existing
			_, err := cfg.Conn.Queries.UpdateMonitor(ctx, &db.UpdateMonitorParams{
				ID:            dbMonitor.ID,
				Name:          configMonitor.Name,
				Url:           configMonitor.URL,
				CheckInterval: configMonitor.CheckInterval,
			})
			if err != nil {
				return err
			}
			slog.Info("Updated monitor from config", "url", url)
		} else {
			// Create new
			_, err := cfg.Conn.Queries.CreateMonitor(ctx, &db.CreateMonitorParams{
				Name:          configMonitor.Name,
				Url:           configMonitor.URL,
				CheckInterval: configMonitor.CheckInterval,
			})
			if err != nil {
				return err
			}
			slog.Info("Added monitor from config", "url", url)
		}
	}

	// Remove monitors not in config
	for url, dbMonitor := range dbMap {
		if _, exists := configMap[url]; !exists {
			err := cfg.Conn.Queries.DeleteMonitor(ctx, dbMonitor.ID)
			if err != nil {
				return err
			}
			slog.Info("Removed monitor not in config", "url", url)
		}
	}

	return nil
}
