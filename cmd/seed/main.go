package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/kata/logx"
	"github.com/mizuchilabs/kata/sigx"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "seed",
		Usage: "Generate random test data for all monitors",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "days",
				Aliases: []string{"n"},
				Usage:   "Days of history to generate",
				Value:   14,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logx.Init(cmd.Bool("debug"))
			return run(ctx, cmd.Int("days"))
		},
	}

	if err := cmd.Run(sigx.NotifyContext(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, days int) error {
	envCfg, err := env.ParseAs[config.EnvConfig]()
	if err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}

	q := db.NewConnection(ctx)
	if err := config.SyncMonitors(ctx, q, envCfg); err != nil {
		return fmt.Errorf("failed to sync monitors: %w", err)
	}

	monitors, err := q.GetMonitors(ctx)
	if err != nil {
		return fmt.Errorf("failed to load monitors: %w", err)
	}
	if len(monitors) == 0 {
		return fmt.Errorf("no monitors found, add some to %s", envCfg.ConfigPath)
	}

	start := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	now := time.Now()

	for i, m := range monitors {
		// Clamp to at least 60s so the data volume stays reasonable
		interval := max(m.CheckInterval, 60)
		count := 0

		for t := start; t.Before(now); t = t.Add(time.Duration(interval) * time.Second) {
			// Align to the interval grid so re-running overwrites instead of duplicating
			params := &db.UpsertCheckParams{
				MonitorID: m.ID,
				CheckedAt: t.Unix() - t.Unix()%interval,
			}
			params.IsUp, params.StatusCode, params.ResponseTime, params.Error = generateCheck(i)

			if err := q.UpsertCheck(ctx, params); err != nil {
				return fmt.Errorf("failed to insert check for %q: %w", m.Url, err)
			}
			count++
		}

		slog.Info("Generated test data", "monitor", m.Name, "checks", count)
	}

	return nil
}

// generateCheck produces a realistic check result. The profile derived from the
// monitor index gives each monitor a different reliability and latency behavior.
func generateCheck(profile int) (up bool, code int64, responseTime int64, errStr *string) {
	var downChance float64
	switch profile % 4 {
	case 0: // Excellent - 99.9% uptime
		downChance = 0.001
	case 1: // Good - 99.5% uptime
		downChance = 0.005
	case 2: // Moderate - 98% uptime
		downChance = 0.02
	case 3: // Problematic - 95% uptime
		downChance = 0.05
	}

	if rand.Float64() < downChance { // #nosec G404
		msg := "connection timeout"
		return false, 0, 0, &msg
	}

	latency := rand.Float64() // #nosec G404
	switch {
	case latency < 0.7: // 70% fast
		responseTime = int64(rand.Intn(80) + 20) // #nosec G404
	case latency < 0.9: // 20% moderate
		responseTime = int64(rand.Intn(150) + 100) // #nosec G404
	case latency < 0.98: // 8% slow
		responseTime = int64(rand.Intn(300) + 250) // #nosec G404
	default: // 2% very slow
		responseTime = int64(rand.Intn(500) + 500) // #nosec G404
	}

	return true, 200, responseTime, nil
}
