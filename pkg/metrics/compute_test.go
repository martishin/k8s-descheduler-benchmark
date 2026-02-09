package metrics

import (
	"math"
	"testing"
	"time"
)

func TestStddev(t *testing.T) {
	values := []float64{1, 2, 3, 4}
	got := stddev(values)
	if math.Abs(got-1.1180) > 0.001 {
		t.Fatalf("unexpected stddev: %f", got)
	}
}

func TestMaxMinRatio(t *testing.T) {
	if r := maxMinRatio([]float64{0, 0}); r != 0 {
		t.Fatalf("expected ratio 0, got %f", r)
	}
	if r := maxMinRatio([]float64{0, 2}); r != -1 {
		t.Fatalf("expected ratio -1 for zero min, got %f", r)
	}
	if r := maxMinRatio([]float64{2, 4}); r != 2 {
		t.Fatalf("expected ratio 2, got %f", r)
	}
}

func TestDeriveSampleBasic(t *testing.T) {
	snap := Snapshot{
		Time: time.Now(),
		Nodes: map[string]NodeStats{
			"n1": {CPUAllocatableMilli: 1000, MemAllocatableBytes: 1000, CPURequestedMilli: 200, MemRequestedBytes: 200},
			"n2": {CPUAllocatableMilli: 1000, MemAllocatableBytes: 1000, CPURequestedMilli: 900, MemRequestedBytes: 900},
		},
		UnschedulablePods: 1,
		TotalPodsCounted:  2,
	}
	sample := DeriveSample(snap)
	if sample.UnschedulablePods != 1 {
		t.Fatalf("expected unschedulable 1, got %d", sample.UnschedulablePods)
	}
	if sample.NodesCount != 2 || sample.PodsCounted != 2 {
		t.Fatalf("unexpected counts: nodes=%d pods=%d", sample.NodesCount, sample.PodsCounted)
	}
}
