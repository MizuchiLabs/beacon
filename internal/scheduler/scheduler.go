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
	monitors      map[int64]*monitorJob
	mu            sync.RWMutex
	wg            sync.WaitGroup
	RetentionDays int
}

type monitorJob struct {
	monitor *db.Monitor
	ticker  *time.Ticker
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
		monitors:      make(map[int64]*monitorJob),
		RetentionDays: retentionDays,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	// Load active monitors
	monitors, err := s.conn.Queries.GetMonitors(ctx)
	if err != nil {
		slog.Error("failed to load monitors", "error", err)
		return err
	}

	// Start monitoring
	for _, monitor := range monitors {
		if monitor == nil {
			continue
		}

		job := &monitorJob{
			monitor: monitor,
			ticker:  time.NewTicker(time.Duration(monitor.CheckInterval) * time.Second),
		}
		s.monitors[monitor.ID] = job
		s.wg.Add(1)
		go s.runMonitor(ctx, job)
	}

	// Start cleanup routine
	s.wg.Add(1)
	go s.cleanupJob(ctx)
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	for _, job := range s.monitors {
		job.ticker.Stop()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Scheduler) runMonitor(ctx context.Context, job *monitorJob) {
	defer s.wg.Done()

	// Immediate first check
	s.performCheck(ctx, job.monitor)

	for {
		select {
		case <-job.ticker.C:
			s.performCheck(ctx, job.monitor)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) performCheck(ctx context.Context, monitor *db.Monitor) {
	result := s.checker.Check(ctx, monitor.Url)
	result.MonitorID = monitor.ID

	// Store check result
	_, err := s.conn.Queries.CreateCheck(ctx, result)
	if err != nil {
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
			if err := s.conn.Queries.CleanupChecks(ctx, &daysStr); err != nil {
				slog.Error("Failed to cleanup old checks", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
