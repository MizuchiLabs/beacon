// Package scheduler provides functionality for scheduling jobs
package scheduler

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

type Scheduler struct {
	conn          *db.Connection
	checker       *checker.Checker
	notifier      *notify.Notifier
	wg            sync.WaitGroup
	RetentionDays int
}

func New(
	conn *db.Connection,
	checker *checker.Checker,
	notifier *notify.Notifier,
	retentionDays int,
) *Scheduler {
	if retentionDays <= 1 {
		retentionDays = 30
	}

	return &Scheduler{
		conn:          conn,
		checker:       checker,
		notifier:      notifier,
		RetentionDays: retentionDays,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	// Load active monitors
	monitors, err := s.conn.Q.GetMonitors(ctx)
	if err != nil {
		slog.Error("failed to load monitors", "error", err)
		return
	}

	// Start monitoring
	for _, monitor := range monitors {
		if monitor == nil {
			continue
		}

		s.wg.Go(func() { s.runMonitor(ctx, monitor) })
	}
	s.wg.Go(func() { s.cleanupJob(ctx) })

	// Wait for shutdown signal
	go func() {
		<-ctx.Done()
		s.wg.Wait()
	}()
}

func (s *Scheduler) runMonitor(ctx context.Context, monitor *db.Monitor) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(monitor.CheckInterval) * time.Second)
	defer ticker.Stop()

	// Immediate first check
	s.performCheck(ctx, monitor)

	for {
		select {
		case <-ticker.C:
			s.performCheck(ctx, monitor)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) performCheck(ctx context.Context, monitor *db.Monitor) {
	// Add timeout to prevent hanging checks
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result := s.checker.Check(checkCtx, monitor.Url)
	result.MonitorID = monitor.ID

	// Store check result
	if err := s.conn.Q.CreateCheck(checkCtx, result); err != nil {
		slog.Error("Failed to store check", "monitor_id", monitor.ID, "error", err)
		return
	}

	if !result.IsUp && result.Error != nil {
		if err := s.notifier.SendMonitorDownNotification(ctx, monitor, *result.Error); err != nil {
			slog.Error(
				"Failed to send monitor down notification",
				"monitor_id",
				monitor.ID,
				"error",
				err,
			)
		}
	}
}

func (s *Scheduler) cleanupJob(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			daysStr := strconv.Itoa(s.RetentionDays)
			if err := s.conn.Q.CleanupChecks(ctx, &daysStr); err != nil {
				slog.Error("Failed to cleanup old checks", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
