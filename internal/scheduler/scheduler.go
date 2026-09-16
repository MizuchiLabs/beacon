// Package scheduler provides functionality for scheduling jobs
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

type Service struct {
	q            *db.Queries
	checker      *checker.Checker
	notifier     *notify.Service
	wg           sync.WaitGroup
	mu           sync.Mutex
	lastUp       map[int64]bool
	lastCertWarn map[int64]int64

	RetentionDays int `env:"BEACON_RETENTION_DAYS" envDefault:"30"`
}

func New(
	q *db.Queries,
	checker *checker.Checker,
	notifier *notify.Service,
) (*Service, error) {
	s, err := env.ParseAs[Service]()
	if err != nil {
		return nil, err
	}

	s.q = q
	s.checker = checker
	s.notifier = notifier
	s.lastUp = make(map[int64]bool)
	s.lastCertWarn = make(map[int64]int64)
	return &s, nil
}

func (s *Service) Start(ctx context.Context) {
	// Load active monitors
	monitors, err := s.q.GetMonitors(ctx)
	if err != nil {
		slog.Error("failed to load monitors", "error", err)
		return
	}

	// Seed the last known state so a restart does not report a recovery for an
	// outage nobody was alerted about
	states, err := s.q.GetLatestChecks(ctx)
	if err != nil {
		slog.Error("failed to load last check states", "error", err)
	} else {
		s.mu.Lock()
		for _, state := range states {
			s.lastUp[state.MonitorID] = state.IsUp
		}
		s.mu.Unlock()
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

func (s *Service) runMonitor(ctx context.Context, monitor *db.Monitor) {
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

func (s *Service) performCheck(ctx context.Context, monitor *db.Monitor) {
	result := s.checker.Check(ctx, monitor.Url)
	checkedAt := time.Now().Unix()

	if err := s.q.UpsertCheck(ctx, &db.UpsertCheckParams{
		MonitorID:     monitor.ID,
		StatusCode:    result.StatusCode,
		ResponseTime:  result.ResponseTime,
		DaysRemaining: result.DaysLeft,
		Error:         result.Error,
		IsUp:          result.IsUp,
		CheckedAt:     checkedAt,
	}); err != nil {
		slog.Error("Failed to store check", "monitor_id", monitor.ID, "error", err)
		return
	}

	// Only notify on up/down transitions, not on every failed check
	if s.recordState(monitor.ID, result.IsUp) {
		if err := s.notifier.SendMonitorNotification(ctx, monitor, result.IsUp, checkReason(result)); err != nil {
			slog.Error("Failed to send monitor notification", "monitor_id", monitor.ID, "error", err)
		}
	}

	s.warnCertificateExpiry(ctx, monitor, result)
}

// checkReason builds the human readable cause sent with a transition
// notification.
func checkReason(result checker.Result) string {
	if result.Error != nil {
		return *result.Error
	}
	switch {
	case result.DaysLeft != nil:
		return fmt.Sprintf("certificate valid, %d days remaining", *result.DaysLeft)
	case result.StatusCode != 0:
		return fmt.Sprintf("HTTP %d", result.StatusCode)
	default:
		return "connection established"
	}
}

// warnCertificateExpiry notifies subscribers once per day while a valid
// certificate is inside the warning window. Expiry is not an up/down
// transition, so the cadence is tracked separately from recordState.
func (s *Service) warnCertificateExpiry(ctx context.Context, monitor *db.Monitor, result checker.Result) {
	if monitor.IgnoreCertExpiry || !result.IsUp || result.DaysLeft == nil ||
		*result.DaysLeft >= checker.CertWarnDays {
		return
	}

	if !s.certWarnDue(monitor.ID) {
		return
	}

	if err := s.notifier.SendCertificateExpiryNotification(ctx, monitor, *result.DaysLeft); err != nil {
		slog.Error("Failed to send certificate expiry notification", "monitor_id", monitor.ID, "error", err)
	}
}

// certWarnDue reports whether monitorID has not had a certificate warning
// today, marking it warned as a side effect.
func (s *Service) certWarnDue(monitorID int64) bool {
	today := time.Now().Unix() / 86400
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastCertWarn[monitorID] == today {
		return false
	}
	s.lastCertWarn[monitorID] = today
	return true
}

// recordState stores the check outcome and reports whether it differs from the
// previously observed state. The first observation never counts as a transition.
func (s *Service) recordState(monitorID int64, isUp bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev, seen := s.lastUp[monitorID]
	s.lastUp[monitorID] = isUp
	return seen && prev != isUp
}

func (s *Service) cleanupJob(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) cleanup(ctx context.Context) {
	if err := s.q.CleanupChecks(ctx, s.cleanupCutoff()); err != nil {
		slog.Error("Failed to cleanup old checks", "error", err)
	}
}

// cleanupCutoff is the oldest check timestamp that survives a cleanup run.
func (s *Service) cleanupCutoff() int64 {
	return time.Now().AddDate(0, 0, -s.RetentionDays).Unix()
}
