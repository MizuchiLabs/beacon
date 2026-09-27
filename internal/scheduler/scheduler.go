// Package scheduler runs the checks of every monitor on its interval and
// turns their results into notifications.
package scheduler

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/notify"
)

const (
	// secondsPerDay is the bucket size for once-per-day notification cadence.
	secondsPerDay = 86400
)

// retryDelay is the pause between attempts of a failing check.
var retryDelay = 5 * time.Second

// ErrUnknownHeartbeat is returned for a heartbeat token no push monitor uses.
var ErrUnknownHeartbeat = errors.New("unknown heartbeat token")

type Service struct {
	q        *db.Queries
	checker  *checker.Checker
	notifier *notify.Service

	mu           sync.Mutex
	stop         context.CancelFunc // stops the goroutines of the current monitor set
	byToken      map[string]*db.Monitor
	lastUp       map[int64]bool
	lastCertWarn map[int64]int64
	lastPing     map[int64]time.Time

	RetentionDays int `env:"BEACON_RETENTION_DAYS" envDefault:"30"`
	HistoryDays   int `env:"BEACON_HISTORY_DAYS"   envDefault:"365"`
}

func New(q *db.Queries, checker *checker.Checker, notifier *notify.Service) (*Service, error) {
	s, err := env.ParseAs[Service]()
	if err != nil {
		return nil, err
	}

	s.q = q
	s.checker = checker
	s.notifier = notifier
	s.byToken = make(map[string]*db.Monitor)
	s.lastUp = make(map[int64]bool)
	s.lastCertWarn = make(map[int64]int64)
	s.lastPing = make(map[int64]time.Time)
	return &s, nil
}

// Start seeds the last known states and runs the cleanup job. Monitors are
// handed over with Schedule.
func (s *Service) Start(ctx context.Context) error {
	// Seeding keeps a restart from reporting a recovery for an outage nobody
	// was alerted about, and gives heartbeats their last ping back.
	states, err := s.q.GetLatestChecks(ctx)
	if err != nil {
		return fmt.Errorf("loading last check states: %w", err)
	}
	s.mu.Lock()
	for _, state := range states {
		s.lastUp[state.MonitorID] = state.IsUp
		if state.IsUp {
			s.lastPing[state.MonitorID] = time.Unix(state.CheckedAt, 0)
		}
	}
	s.mu.Unlock()

	go s.cleanupJob(ctx)
	return nil
}

// Schedule stops the checks of the previous monitor set and starts checking
// monitors instead. It is called again whenever the config changes.
func (s *Service) Schedule(ctx context.Context, monitors []*db.Monitor) {
	s.mu.Lock()
	if s.stop != nil {
		s.stop()
	}
	ctx, s.stop = context.WithCancel(ctx)
	clear(s.byToken)
	for _, m := range monitors {
		if m.Type != checker.TypePush {
			continue
		}
		s.byToken[pushToken(m)] = m
		// A new heartbeat gets one full interval before it counts as missed.
		if _, ok := s.lastPing[m.ID]; !ok {
			s.lastPing[m.ID] = time.Now()
		}
	}
	s.mu.Unlock()

	for _, m := range monitors {
		go s.runMonitor(ctx, m)
	}
}

// Heartbeat records a ping for the push monitor that owns token. A ping with
// up false reports a failure, message says why.
func (s *Service) Heartbeat(ctx context.Context, token string, up bool, message string) error {
	s.mu.Lock()
	m, ok := s.byToken[token]
	if ok && up {
		s.lastPing[m.ID] = time.Now()
	}
	s.mu.Unlock()
	if !ok {
		return ErrUnknownHeartbeat
	}

	// The ping request may hang up before the notifications are out.
	ctx = context.WithoutCancel(ctx)
	result := checker.Result{IsUp: up}
	if !up {
		result.Error = new(cmp.Or(message, "heartbeat reported a failure"))
	}
	return s.record(ctx, m, result)
}

func (s *Service) runMonitor(ctx context.Context, m *db.Monitor) {
	interval := time.Duration(m.CheckInterval) * time.Second
	if m.Type != checker.TypePush {
		s.check(ctx, m)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if m.Type == checker.TypePush {
			s.checkHeartbeat(ctx, m, interval)
		} else {
			s.check(ctx, m)
		}
	}
}

// check runs the monitor's check, retrying a failure before it counts.
func (s *Service) check(ctx context.Context, m *db.Monitor) {
	result := s.checker.Check(ctx, m)
	for range m.Retries {
		if result.IsUp {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
		result = s.checker.Check(ctx, m)
	}

	// A check cut short by a reload or shutdown says nothing about the target.
	if ctx.Err() != nil {
		return
	}
	if err := s.record(ctx, m, result); err != nil {
		slog.Error("Failed to store check", "monitor", m.Name, "error", err)
	}
}

// checkHeartbeat marks a push monitor down once its ping is overdue. Half an
// interval of slack keeps a slightly late cron job from flapping.
func (s *Service) checkHeartbeat(ctx context.Context, m *db.Monitor, interval time.Duration) {
	s.mu.Lock()
	since := time.Since(s.lastPing[m.ID])
	s.mu.Unlock()
	if since <= interval+interval/2 {
		return
	}

	msg := fmt.Sprintf("no heartbeat for %s", since.Round(time.Second))
	if err := s.record(ctx, m, checker.Result{Error: &msg}); err != nil {
		slog.Error("Failed to store check", "monitor", m.Name, "error", err)
	}
}

// record stores a result and sends the notifications it calls for.
func (s *Service) record(ctx context.Context, m *db.Monitor, result checker.Result) error {
	if err := s.q.InsertCheck(ctx, &db.InsertCheckParams{
		MonitorID:     m.ID,
		StatusCode:    result.StatusCode,
		ResponseTime:  result.ResponseTime,
		DaysRemaining: result.DaysLeft,
		Error:         result.Error,
		IsUp:          result.IsUp,
		CheckedAt:     time.Now().Unix(),
	}); err != nil {
		return err
	}

	if s.transitioned(m.ID, result.IsUp) {
		if result.IsUp {
			s.notifier.MonitorUp(ctx, m)
		} else {
			s.notifier.MonitorDown(ctx, m, reason(result))
		}
	}

	if !m.IgnoreCertExpiry && result.IsUp && result.DaysLeft != nil &&
		*result.DaysLeft < checker.CertWarnDays && s.certWarnDue(m.ID) {
		s.notifier.CertExpiring(ctx, m, *result.DaysLeft)
	}
	return nil
}

// reason builds the human readable cause sent with a down notification.
func reason(result checker.Result) string {
	if result.Error != nil {
		return *result.Error
	}
	return fmt.Sprintf("HTTP %d", result.StatusCode)
}

// transitioned stores the check outcome and reports whether it differs from
// the previous one. The first observation never counts as a transition.
func (s *Service) transitioned(monitorID int64, isUp bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev, seen := s.lastUp[monitorID]
	s.lastUp[monitorID] = isUp
	return seen && prev != isUp
}

// certWarnDue reports whether monitorID has not had a certificate warning
// today, marking it warned as a side effect.
func (s *Service) certWarnDue(monitorID int64) bool {
	today := time.Now().Unix() / secondsPerDay
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastCertWarn[monitorID] == today {
		return false
	}
	s.lastCertWarn[monitorID] = today
	return true
}

func (s *Service) cleanupJob(ctx context.Context) {
	tick := time.Tick(time.Hour)
	for {
		s.cleanup(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick:
		}
	}
}

func (s *Service) cleanup(ctx context.Context) {
	checksCutoff, rollupsCutoff := s.cutoffs()
	if err := s.q.CleanupChecks(ctx, checksCutoff); err != nil {
		slog.Error("Failed to clean up old checks", "error", err)
	}
	if err := s.q.CleanupRollups(ctx, rollupsCutoff); err != nil {
		slog.Error("Failed to clean up old rollups", "error", err)
	}
}

// cutoffs are the oldest raw check and hourly rollup that survive a cleanup.
func (s *Service) cutoffs() (checks, rollups int64) {
	now := time.Now()
	return now.AddDate(0, 0, -s.RetentionDays).Unix(), now.AddDate(0, 0, -s.HistoryDays).Unix()
}

// pushToken is the secret part of a push url, push://<token>.
func pushToken(m *db.Monitor) string {
	u, err := url.Parse(m.Url)
	if err != nil {
		return ""
	}
	return u.Host
}
