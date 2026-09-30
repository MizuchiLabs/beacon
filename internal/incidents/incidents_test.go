package incidents

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncRepo(t *testing.T) {
	t.Parallel()

	remoteDir := t.TempDir()
	remote, err := git.PlainInit(remoteDir, false)
	require.NoError(t, err)
	wt, err := remote.Worktree()
	require.NoError(t, err)

	commit := func(name, title string, amend bool) {
		t.Helper()
		content := "title: " + title + "\nseverity: minor\nstatus: resolved\nstarted_at: 2025-01-01T00:00:00Z\n"
		require.NoError(t, os.WriteFile(filepath.Join(remoteDir, name), []byte(content), 0o600))
		_, err := wt.Add(name)
		require.NoError(t, err)
		_, err = wt.Commit(title, &git.CommitOptions{
			Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
			Amend:  amend,
		})
		require.NoError(t, err)
	}
	titles := func(s *Service) []string {
		t.Helper()
		require.NoError(t, s.syncRepo(t.Context()))
		_, err := s.loadIncidents()
		require.NoError(t, err)
		var out []string
		for _, inc := range s.GetIncidents() {
			out = append(out, inc.Title)
		}
		return out
	}

	commit("first.yaml", "First", false)
	s := &Service{RepoURL: remoteDir, RepoPath: filepath.Join(t.TempDir(), "incidents")}
	assert.Equal(t, []string{"First"}, titles(s), "initial clone")

	commit("second.yaml", "Second", false)
	assert.ElementsMatch(t, []string{"First", "Second"}, titles(s), "new commit is fetched")

	commit("second.yaml", "Rewritten", true)
	assert.ElementsMatch(t, []string{"First", "Rewritten"}, titles(s), "force push is followed")

	assert.ElementsMatch(t, []string{"First", "Rewritten"}, titles(s), "sync without changes is a no-op")
}

func TestChanges(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	update := func(status string, at time.Time) IncidentUpdate {
		return IncidentUpdate{Message: status, Status: status, CreatedAt: at}
	}
	before := []Incident{
		{
			ID:        "old",
			StartedAt: now.Add(-2 * time.Hour),
			Updates:   []IncidentUpdate{update("investigating", now.Add(-2*time.Hour))},
		},
	}
	after := []Incident{
		{ID: "old", StartedAt: now.Add(-2 * time.Hour), Updates: []IncidentUpdate{
			update("investigating", now.Add(-2*time.Hour)),
			update("identified", now.Add(-time.Minute)),
		}},
		{ID: "fresh", Status: "investigating", StartedAt: now.Add(-time.Minute)},
		{ID: "history", Status: "resolved", StartedAt: now.Add(-48 * time.Hour)},
		{
			ID:        "maint",
			StartedAt: now.Add(-3 * time.Hour),
			Updates:   []IncidentUpdate{update("scheduled", now.Add(-3*time.Hour))},
		},
	}

	var got []string
	for _, e := range changes(before, after, now) {
		got = append(got, e.ID+":"+e.Status)
	}
	assert.Equal(t, []string{"old:identified", "fresh:investigating", "maint:scheduled"}, got,
		"new updates and incidents notify, backfilled history does not, announced maintenance always does")
	assert.Empty(t, changes(after, after, now), "an unchanged reload is quiet")
}
