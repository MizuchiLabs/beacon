package incidents

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIncidentsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	write("old.yaml", "title: Old\nseverity: minor\nstatus: resolved\nstarted_at: 2025-01-01T00:00:00Z\n")
	write(
		"new.yml",
		"id: custom\ntitle: New\nseverity: major\nstatus: investigating\nstarted_at: 2025-02-01T00:00:00Z\n",
	)
	write("typo.yaml", "title: Typo\nseverity: majr\nstatus: resolved\nstarted_at: 2025-03-01T00:00:00Z\n")
	write("notes.txt", "not an incident")

	incidents, err := ParseIncidentsDir(dir)
	require.NoError(t, err)
	require.Len(t, incidents, 2, "invalid files and other extensions are skipped")

	assert.Equal(t, "custom", incidents[0].ID, "newest first")
	assert.Equal(t, "old", incidents[1].ID, "the file name is the default id")
}

func TestDerive(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, content string) Incident {
		t.Helper()
		inc, err := parseBytes([]byte(content), "x.yaml")
		require.NoError(t, err)
		return inc
	}
	const updates = "updates:\n" +
		"  - {message: Fixed, status: resolved, created_at: 2025-01-01T02:00:00Z}\n" +
		"  - {message: Looking, status: investigating, created_at: 2025-01-01T00:00:00Z}\n"

	t.Run("status and resolved_at come from the latest update", func(t *testing.T) {
		t.Parallel()
		inc := parse(t, "title: A\nseverity: major\nstarted_at: 2025-01-01T00:00:00Z\n"+updates)
		assert.Equal(t, "resolved", inc.Status)
		require.NotNil(t, inc.ResolvedAt)
		assert.Equal(t, "2025-01-01T02:00:00Z", inc.ResolvedAt.Format(time.RFC3339))
		assert.Equal(t, "Looking", inc.Updates[0].Message, "updates are sorted oldest first")
	})

	t.Run("fields in the file win", func(t *testing.T) {
		t.Parallel()
		inc := parse(t, "title: A\nseverity: major\nstatus: monitoring\nstarted_at: 2025-01-01T00:00:00Z\n"+updates)
		assert.Equal(t, "monitoring", inc.Status)
		assert.Nil(t, inc.ResolvedAt)
	})

	t.Run("defaults without updates", func(t *testing.T) {
		t.Parallel()
		assert.Equal(
			t,
			"investigating",
			parse(t, "title: A\nseverity: minor\nstarted_at: 2025-01-01T00:00:00Z\n").Status,
		)
		assert.Equal(
			t,
			"scheduled",
			parse(t, "title: A\nseverity: maintenance\nstarted_at: 2025-01-01T00:00:00Z\n").Status,
		)
	})

	t.Run("invalid files", func(t *testing.T) {
		t.Parallel()
		for name, content := range map[string]string{
			"no title":         "severity: minor\nstarted_at: 2025-01-01T00:00:00Z\n",
			"no start":         "title: A\nseverity: minor\n",
			"end before start": "title: A\nseverity: maintenance\nstarted_at: 2025-01-01T00:00:00Z\nends_at: 2024-12-31T00:00:00Z\n",
			"bad update":       "title: A\nseverity: minor\nstarted_at: 2025-01-01T00:00:00Z\nupdates: [{message: x, status: done, created_at: 2025-01-01T00:00:00Z}]\n",
		} {
			_, err := parseBytes([]byte(content), "x.yaml")
			assert.Error(t, err, name)
		}
	})
}

func TestMaintenanceEndsItself(t *testing.T) {
	t.Parallel()

	start := time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC)
	inc := Incident{Status: "scheduled", StartedAt: start, EndsAt: new(start.Add(2 * time.Hour))}

	assert.Equal(t, "scheduled", inc.at(start.Add(time.Hour)).Status, "inside the window")
	ended := inc.at(start.Add(3 * time.Hour))
	assert.Equal(t, "resolved", ended.Status, "after the window")
	assert.Equal(t, inc.EndsAt, ended.ResolvedAt)
	assert.Equal(t, "scheduled", inc.Status, "the stored incident is untouched")
}

func TestDuplicateIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		content := "id: same\ntitle: T\nseverity: minor\nstarted_at: 2025-01-01T00:00:00Z\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	incidents, errs, err := parseDir(dir)
	require.NoError(t, err)
	assert.Len(t, incidents, 1)
	assert.Len(t, errs, 1)
}
