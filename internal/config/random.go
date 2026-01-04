package config

import (
	"context"
	"log/slog"
	"math/rand"
	"time"
)

func (c *Config) GenerateRandomData(ctx context.Context, gen bool) {
	if !gen {
		return
	}
	monitors, err := c.Conn.Queries.GetMonitors(ctx)
	if err != nil {
		slog.Error("Failed to get monitors", "error", err)
		return
	}

	if len(monitors) == 0 {
		slog.Warn("No monitors found, skipping random data generation")
		return
	}

	now := time.Now()
	days := 14
	start := now.Add(-time.Duration(days) * 24 * time.Hour)

	slog.Info("Generating random test data", "monitors", len(monitors), "days", days)
	for _, m := range monitors {
		interval := max(int(m.CheckInterval), 60)

		t := start
		for t.Before(now) {
			up, respTime := generateRealisticCheck()

			code := int64(200)
			var errStr *string
			if !up {
				errMsg := "connection timeout"
				errStr = &errMsg
				code = 0
			}

			_, err := c.Conn.Get().ExecContext(ctx, `
				INSERT INTO checks (monitor_id, status_code, response_time, error, is_up, checked_at)
				VALUES (?, ?, ?, ?, ?, datetime(?, 'unixepoch'))`,
				m.ID, code, respTime, errStr, up, t.Unix(),
			)
			if err != nil {
				slog.Error("Failed to insert check", "monitor_id", m.ID, "error", err)
			}

			t = t.Add(time.Duration(interval) * time.Second)
		}
	}
}

func generateRealisticCheck() (up bool, respTime int64) {
	r := rand.Float64() // #nosec G404

	switch {
	case r < 0.02:
		up = false
		respTime = int64(rand.Intn(5000) + 3000) // #nosec G404
	case r < 0.05:
		up = true
		respTime = int64(rand.Intn(400) + 600) // #nosec G404
	case r < 0.15:
		up = true
		respTime = int64(rand.Intn(200) + 300) // #nosec G404
	default:
		up = true
		respTime = int64(rand.Intn(150) + 50) // #nosec G404
	}
	return
}
