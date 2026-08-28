// Package scheduler provides functionality for scheduling jobs
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

type Scheduler struct {
	q             *db.Queries
	checker       *checker.Checker
	notifier      *notify.Notifier
	retentionDays int
	wg            sync.WaitGroup

	mu     sync.Mutex
	lastUp map[int64]bool
}

func New(
	q *db.Queries,
	checker *checker.Checker,
	notifier *notify.Notifier,
	retentionDays int,
) *Scheduler {
	if retentionDays <= 1 {
		retentionDays = 30
	}

	return &Scheduler{
		q:             q,
		checker:       checker,
		notifier:      notifier,
		retentionDays: retentionDays,
		lastUp:        make(map[int64]bool),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	// Load active monitors
	monitors, err := s.q.GetMonitors(ctx)
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
}

func (s *Scheduler) runMonitor(ctx context.Context, monitor *db.Monitor) {
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
	result := s.checker.Check(ctx, monitor.Url)
	checkedAt := time.Now().Unix()

	if err := s.q.UpsertCheck(ctx, &db.UpsertCheckParams{
		MonitorID:    monitor.ID,
		StatusCode:   result.StatusCode,
		ResponseTime: result.ResponseTime,
		Error:        result.Error,
		IsUp:         result.IsUp,
		CheckedAt:    checkedAt,
	}); err != nil {
		slog.Error("Failed to store check", "monitor_id", monitor.ID, "error", err)
		return
	}

	// Only notify on up/down transitions, not on every failed check
	if !s.recordState(monitor.ID, result.IsUp) {
		return
	}

	reason := fmt.Sprintf("HTTP %d", result.StatusCode)
	if result.Error != nil {
		reason = *result.Error
	}

	if err := s.notifier.SendMonitorNotification(ctx, monitor, result.IsUp, reason); err != nil {
		slog.Error("Failed to send monitor notification", "monitor_id", monitor.ID, "error", err)
	}
}

// recordState stores the check outcome and reports whether it differs from the
// previously observed state. The first observation never counts as a transition.
func (s *Scheduler) recordState(monitorID int64, isUp bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev, seen := s.lastUp[monitorID]
	s.lastUp[monitorID] = isUp
	return seen && prev != isUp
}

func (s *Scheduler) cleanupJob(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -s.retentionDays).Unix()
			if err := s.q.CleanupChecks(ctx, cutoff); err != nil {
				slog.Error("Failed to cleanup old checks", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
