package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/mizuchilabs/kata/logx"
	"github.com/mizuchilabs/kata/sigx"

	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
)

func main() {
	cmd := &cli.Command{
		Name:  "seed",
		Usage: "Generate random test data for all monitors",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("BEACON_DEBUG"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Usage:   "Directory for the database",
				Value:   "data",
				Sources: cli.EnvVars("BEACON_DATA_DIR"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to monitors config file",
				Value:   "config.yaml",
				Sources: cli.EnvVars("BEACON_CONFIG"),
			},
			&cli.IntFlag{
				Name:    "days",
				Aliases: []string{"n"},
				Usage:   "Days of history to generate",
				Value:   14,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logx.Init(cmd.Bool("debug"))
			return run(ctx, cmd)
		},
	}

	if err := cmd.Run(sigx.NotifyContext(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	q, err := db.Open(ctx, cmd.String("data-dir"))
	if err != nil {
		return err
	}

	cfg, err := config.Load(cmd.String("config"))
	if err != nil {
		return err
	}
	monitors, err := config.Sync(ctx, q, cfg.Monitors)
	if err != nil {
		return fmt.Errorf("failed to sync monitors: %w", err)
	}
	if len(monitors) == 0 {
		return fmt.Errorf("no monitors found, add some to %s", cmd.String("config"))
	}

	start := time.Now().Add(-time.Duration(cmd.Int("days")) * 24 * time.Hour)
	now := time.Now()

	for i, m := range monitors {
		// Clamp to at least 60s so the data volume stays reasonable
		interval := max(m.CheckInterval, 60)
		bad := badSpells(i, start, now)
		count := 0

		for t := start; t.Before(now); t = t.Add(time.Duration(interval) * time.Second) {
			// Align to the interval grid so re-running skips existing checks
			params := &db.InsertCheckParams{
				MonitorID: m.ID,
				CheckedAt: t.Unix() - t.Unix()%interval,
			}
			params.IsUp, params.StatusCode, params.ResponseTime, params.Error = generateCheck(params.CheckedAt, bad)

			if err := q.InsertCheck(ctx, params); err != nil {
				return fmt.Errorf("failed to insert check for %q: %w", m.Url, err)
			}
			count++
		}

		slog.Info("Generated test data", "monitor", m.Name, "checks", count)
	}

	return nil
}

// spell is a stretch of time where a monitor is down or slow.
type spell struct {
	from, to int64
	slow     bool
}

// badSpells picks the rough patches of one monitor. Real outages come in runs,
// a coin flip per check would sprinkle failures evenly over the whole history.
// The profile derived from the monitor index gives each monitor a different reliability.
func badSpells(profile int, start, end time.Time) []spell {
	perWeek := []float64{0, 0.5, 2, 4}[profile%4]
	weeks := end.Sub(start).Hours() / (7 * 24)

	spells := make([]spell, int(perWeek*weeks+rand.Float64())) //nolint:gosec // synthetic seed data
	for i := range spells {
		from := start.Unix() + rand.Int64N(end.Unix()-start.Unix()) //nolint:gosec // synthetic seed data
		minutes := rand.Int64N(40) + 3                              //nolint:gosec // synthetic seed data
		slow := rand.IntN(2) == 0                                   //nolint:gosec // synthetic seed data
		if slow {
			minutes *= 2
		}
		spells[i] = spell{from: from, to: from + minutes*60, slow: slow}
	}
	return spells
}

// generateCheck produces a realistic check result for a check at ts.
func generateCheck(ts int64, bad []spell) (up bool, code int64, responseTime int64, errStr *string) {
	for _, s := range bad {
		if ts < s.from || ts >= s.to {
			continue
		}
		if s.slow {
			return true, 200, int64(rand.IntN(700) + 600), nil //nolint:gosec // synthetic seed data
		}
		return false, 0, 0, new("connection timeout")
	}

	latency := rand.Float64() //nolint:gosec // synthetic seed data
	switch {
	case latency < 0.7: // 70% fast
		responseTime = int64(rand.IntN(80) + 20) //nolint:gosec // synthetic seed data
	case latency < 0.95: // 25% moderate
		responseTime = int64(rand.IntN(150) + 100) //nolint:gosec // synthetic seed data
	default: // 5% slow, still under the default degraded threshold
		responseTime = int64(rand.IntN(200) + 250) //nolint:gosec // synthetic seed data
	}

	return true, 200, responseTime, nil
}
