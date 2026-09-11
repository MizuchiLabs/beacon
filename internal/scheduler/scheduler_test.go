package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecordStateReportsOnlyTransitions(t *testing.T) {
	t.Parallel()

	s := &Scheduler{lastUp: make(map[int64]bool)}

	require.False(t, s.recordState(1, true), "first observation is not a transition")
	require.False(t, s.recordState(1, true))
	require.True(t, s.recordState(1, false), "up to down is a transition")
	require.False(t, s.recordState(1, false))
	require.True(t, s.recordState(1, true), "down to up is a transition")
	require.False(t, s.recordState(2, true), "state is tracked per monitor")
}

func TestCleanupCutoffUsesConfiguredRetention(t *testing.T) {
	t.Setenv("BEACON_RETENTION_DAYS", "7")

	s, err := New(nil, nil, nil)
	require.NoError(t, err)

	require.WithinDuration(
		t,
		time.Now().AddDate(0, 0, -7),
		time.Unix(s.cleanupCutoff(), 0),
		time.Minute,
		"cleanup must delete on the configured retention window",
	)
}

func TestRetentionFallsBackForNonsense(t *testing.T) {
	t.Setenv("BEACON_RETENTION_DAYS", "0")

	s, err := New(nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, defaultRetentionDays, s.RetentionDays)
}
