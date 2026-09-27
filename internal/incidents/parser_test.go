package incidents

import (
	"os"
	"path/filepath"
	"testing"

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
