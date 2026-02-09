package benchmark

import (
	"testing"
	"time"

	"k8s-descheduler-benchmark/internal/metrics"
	"k8s-descheduler-benchmark/internal/report"
)

func TestComputeRebalanceTime(t *testing.T) {
	start := time.Now()
	phases := []report.PhaseMarker{{Name: "uncordon:done", Time: start}}
	samples := []metrics.Sample{
		{Time: start.Add(10 * time.Second), PodsStddev: 5},
		{Time: start.Add(20 * time.Second), PodsStddev: 0.5},
	}
	seconds := computeRebalanceTime(samples, phases, 1.0)
	if seconds < 19 || seconds > 21 {
		t.Fatalf("expected ~20s, got %f", seconds)
	}
}
