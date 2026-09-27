package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadgeSVGEscapesText(t *testing.T) {
	t.Parallel()

	svg := string(badgeSVG(`<script>"x"</script>`, "up", "#fff"))
	assert.NotContains(t, svg, "<script>")
	assert.Contains(t, svg, "&lt;script&gt;")
}

func TestUptimeBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		alive, total int64
		value, color string
	}{
		{0, 0, "no data", badgeColors[statusUnknown]},
		{1000, 1000, "100.00%", badgeColors[statusOperational]},
		{97, 100, "97.00%", badgeColors[statusDegraded]},
		{50, 100, "50.00%", badgeColors[statusDown]},
	}
	for _, tt := range tests {
		value, color := uptimeBadge(tt.alive, tt.total)
		assert.Equal(t, tt.value, value)
		assert.Equal(t, tt.color, color)
	}
}
