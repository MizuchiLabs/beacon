package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
)

type Scheduler struct {
	q               *db.Queries
	checker         *checker.Checker
	monitors        map[int64]*monitorJob
	mu              sync.RWMutex
	wg              sync.WaitGroup
	incidentTracker *incidentTracker
}

type monitorJob struct {
	monitor *db.Monitor
	ticker  *time.Ticker
}

func New(q *db.Queries, checker *checker.Checker) *Scheduler {
	return &Scheduler{
		q:               q,
		checker:         checker,
		monitors:        make(map[int64]*monitorJob),
		incidentTracker: newIncidentTracker(q),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	// Load active monitors
	monitors, err := s.q.GetMonitors(ctx)
	if err != nil {
		return fmt.Errorf("failed to load monitors: %w", err)
	}

	// Start monitoring
	for _, monitor := range monitors {
		if monitor == nil {
			continue
		}

		s.monitors[monitor.ID] = &monitorJob{
			monitor: monitor,
			ticker:  time.NewTicker(time.Duration(monitor.CheckInterval) * time.Second),
		}
	}

	slog.Info("Scheduler started", "monitors", len(monitors))
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	for _, job := range s.monitors {
		job.ticker.Stop()
	}
	s.mu.Unlock()

	s.wg.Wait()
	s.checker.Close()
	slog.Info("Scheduler stopped")
}

func (s *Scheduler) AddMonitor(ctx context.Context, monitor db.Monitor) {
	// Check if already exists
	if _, exists := s.monitors[monitor.ID]; exists {
		slog.Warn("Monitor already scheduled", "id", monitor.ID)
		return
	}

	interval := time.Duration(monitor.CheckInterval) * time.Second
	ticker := time.NewTicker(interval)

	newMonitor, err := s.q.CreateMonitor(ctx, &db.CreateMonitorParams{
		Name:          monitor.Name,
		Url:           monitor.Url,
		CheckInterval: int64(interval.Seconds()),
		IsActive:      true,
	})
	if err != nil {
		slog.Error("Failed to add monitor", "error", err)
		return
	}

	job := &monitorJob{
		monitor: newMonitor,
		ticker:  ticker,
	}
	s.monitors[monitor.ID] = job
	s.wg.Add(1)

	go s.runMonitor(ctx, job)
	slog.Info("Monitor scheduled", "id", monitor.ID, "url", monitor.Url, "interval", interval)
}

func (s *Scheduler) DeleteMonitor(ctx context.Context, monitorID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.monitors[monitorID]; exists {
		job.ticker.Stop()
		delete(s.monitors, monitorID)

		if err := s.q.DeleteMonitor(ctx, monitorID); err != nil {
			slog.Error("Failed to delete monitor", "id", monitorID, "error", err)
		}
		slog.Info("Monitor removed", "id", monitorID)
	}
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

	// Store check result
	check, err := s.q.CreateCheck(ctx, result)
	if err != nil {
		slog.Error("Failed to store check", "monitor_id", monitor.ID, "error", err)
		return
	}

	// Track incidents
	s.incidentTracker.Track(ctx, monitor.ID, result.IsUp, *result.Error)

	slog.Debug("check completed",
		"monitor_id", monitor.ID,
		"url", monitor.Url,
		"is_up", result.IsUp,
		"status", result.StatusCode,
		"response_time", result.ResponseTime,
		"check_id", check.ID,
	)
}
