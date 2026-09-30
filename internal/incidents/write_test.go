package incidents

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndUpdate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	start := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
	path, err := Create(dir, Incident{Title: "Database: slow queries!", Severity: "major", StartedAt: start})
	require.NoError(t, err)
	assert.Equal(t, "2025-01-15-database-slow-queries.yaml", filepath.Base(path))

	_, err = Create(dir, Incident{Title: "Database: slow queries!", Severity: "major", StartedAt: start})
	require.Error(t, err, "an existing file is never overwritten")

	_, err = AddUpdate(dir, "2025-01-15-database-slow-queries", IncidentUpdate{
		Message: "Fixed", Status: "resolved", CreatedAt: start.Add(time.Hour),
	})
	require.NoError(t, err)

	inc, err := parseIncident(path)
	require.NoError(t, err)
	assert.Equal(t, "resolved", inc.Status)
	require.NotNil(t, inc.ResolvedAt)
	assert.Equal(t, start.Add(time.Hour), *inc.ResolvedAt)
}

func TestAddUpdateKeepsFileInStep(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "# on-call notes stay\nid: db\ntitle: DB\nseverity: minor\nstatus: investigating\nstarted_at: 2025-01-01T00:00:00Z\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "whatever.yaml"), []byte(content), 0o600))

	path, err := AddUpdate(dir, "db", IncidentUpdate{Message: "Found it", Status: "identified", CreatedAt: time.Now()})
	require.NoError(t, err, "a file is found by the id it sets")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# on-call notes stay")

	inc, err := parseIncident(path)
	require.NoError(t, err)
	assert.Equal(t, "identified", inc.Status, "a hand set status follows the new update")
	assert.Len(t, inc.Updates, 1)

	_, err = AddUpdate(dir, "nope", IncidentUpdate{Message: "x", Status: "resolved", CreatedAt: time.Now()})
	require.Error(t, err)
	_, err = AddUpdate(dir, "db", IncidentUpdate{Message: "x", Status: "done", CreatedAt: time.Now()})
	assert.Error(t, err)
}

func TestSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "api-is-down", slug("  API is DOWN!!"))
	assert.Equal(t, "störung-im-rz", slug("Störung im RZ"))
	assert.Equal(t, "incident", slug("!!!"))
	assert.Len(t, []rune(slug("a very long title that keeps going and going and going on")), 50)
}
