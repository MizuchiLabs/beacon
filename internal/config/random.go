package config

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

// only for testing purposes
func (c *Config) GenerateRandomData(ctx context.Context) {
	monitors, err := c.Conn.Queries.GetMonitors(ctx)
	if err != nil {
		slog.Error("Failed to get monitors", "error", err)
		return
	}

	for _, m := range monitors {
		start := time.Now().Add(-7 * 24 * time.Hour)
		for t := start; t.Before(time.Now()); t = t.Add(time.Duration(m.CheckInterval) * time.Second) {

			up := rand.Intn(20) != 0 // ~5% down
			var code sql.NullInt64
			var errStr sql.NullString

			if up {
				code.Valid = true
				code.Int64 = 200
			} else {
				errStr.Valid = true
				errStr.String = "timeout"
			}

			resp := rand.Intn(900) + 50

			_, err := c.Conn.Get().Exec(`
				INSERT INTO checks (monitor_id, status_code, response_time, error, is_up, checked_at)
				VALUES (?, ?, ?, ?, ?, ?)`,
				m.ID, code, resp, errStr, up, t,
			)
			if err != nil {
				fmt.Println("insert error:", err)
			}
		}
	}
}
