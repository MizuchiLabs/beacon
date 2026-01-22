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
	monitors, err := c.Conn.Q.GetMonitors(ctx)
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

	tx, err := c.Conn.Get().BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO checks (monitor_id, status_code, response_time, error, is_up, checked_at)
		VALUES (?, ?, ?, ?, ?, datetime(?, 'unixepoch'))`)
	if err != nil {
		slog.Error("Failed to prepare statement", "error", err)
		return
	}
	defer stmt.Close()

	for i, m := range monitors {
		interval := max(int(m.CheckInterval), 60)

		t := start
		for t.Before(now) {
			up, respTime := generateRealisticCheck(i)

			code := int64(200)
			var errStr *string
			if !up {
				errMsg := "connection timeout"
				errStr = &errMsg
				code = 0
			}

			_, err := stmt.ExecContext(ctx, m.ID, code, respTime, errStr, up, t.Unix())
			if err != nil {
				slog.Error("Failed to insert check", "monitor_id", m.ID, "error", err)
			}

			t = t.Add(time.Duration(interval) * time.Second)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
	}
}

func generateRealisticCheck(profile int) (up bool, respTime int64) {
	r := rand.Float64() // #nosec G404

	// Different reliability profiles based on monitor
	var downChance float64
	switch profile % 4 {
	case 0: // Excellent - 99.9%+ uptime
		downChance = 0.001
	case 1: // Good - 99.5% uptime
		downChance = 0.005
	case 2: // Moderate - 98% uptime (will show degraded)
		downChance = 0.02
	case 3: // Problematic - 95% uptime
		downChance = 0.05
	}

	if r < downChance {
		up = false
		respTime = 0 // timeout, no response
		return
	}

	up = true
	// Response time distribution
	latency := rand.Float64() // #nosec G404
	switch {
	case latency < 0.7: // 70% fast
		respTime = int64(rand.Intn(80) + 20) // 20-100ms
	case latency < 0.9: // 20% moderate
		respTime = int64(rand.Intn(150) + 100) // 100-250ms
	case latency < 0.98: // 8% slow
		respTime = int64(rand.Intn(300) + 250) // 250-550ms
	default: // 2% very slow
		respTime = int64(rand.Intn(500) + 500) // 500-1000ms
	}
	return
}
