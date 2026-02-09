package benchmark

import (
	"time"

	"k8s-descheduler-benchmark/pkg/metrics"
	"k8s-descheduler-benchmark/pkg/report"
)

func computeRebalanceTime(samples []metrics.Sample, phases []report.PhaseMarker, threshold float64) float64 {
	if threshold <= 0 {
		return -1
	}
	start := findPhaseTime(phases, "uncordon:done")
	if start.IsZero() {
		return -1
	}
	for _, sample := range samples {
		if sample.Time.Before(start) {
			continue
		}
		if sample.PodsStddev <= threshold {
			return sample.Time.Sub(start).Seconds()
		}
	}
	return -1
}

func findPhaseTime(phases []report.PhaseMarker, name string) time.Time {
	for _, phase := range phases {
		if phase.Name == name {
			return phase.Time
		}
	}
	return time.Time{}
}
