package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		p      int
		sorted []int64
		want   int64
	}{
		{"single value", 50, []int64{7}, 7},
		{"single value at p99", 99, []int64{7}, 7},
		{"median", 50, []int64{10, 20, 30, 40}, 20},
		{"p90 of ten", 90, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 9},
		{"p99 of ten", 99, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, percentile(tt.sorted, tt.p))
		})
	}
}

func TestComputeBucketSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		chartType string
		seconds   int64
		want      int64
	}{
		{"area one day", "area", 86400, 1800},
		{"area one week", "area", 604800, 14400},
		{"area two weeks", "area", 1209600, 28800},
		{"area one month", "area", 2592000, 86400},
		{"bars one day", "bars", 86400, 1080},
		{"bars short window", "bars", 60, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &MonitorService{cfg: &Config{ChartType: tt.chartType}}
			require.Equal(t, tt.want, svc.computeBucketSize(tt.seconds))
		})
	}
}
