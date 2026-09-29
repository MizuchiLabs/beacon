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
		require.NoError(t, s.loadIncidents())
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
