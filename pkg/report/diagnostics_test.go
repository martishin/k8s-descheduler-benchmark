package report

import (
	"testing"

	"k8s-descheduler-benchmark/pkg/metrics"
)

func TestFormatNodePods(t *testing.T) {
	snap := metrics.Snapshot{Nodes: map[string]metrics.NodeStats{
		"b": {Pods: 2},
		"a": {Pods: 1},
		"c": {Pods: 3},
	}}
	got := FormatNodePods(snap)
	want := "a=1 b=2 c=3"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	empty := FormatNodePods(metrics.Snapshot{Nodes: map[string]metrics.NodeStats{}})
	if empty != "-" {
		t.Fatalf("expected '-', got %q", empty)
	}
}

func TestFormatScheduleMessages(t *testing.T) {
	messages := map[string]int{
		"A": 1,
		"B": 3,
		"C": 2,
		"D": 5,
	}
	got := FormatScheduleMessages(messages)
	want := "D (x5); B (x3); C (x2)"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	if FormatScheduleMessages(map[string]int{}) != "none" {
		t.Fatalf("expected none for empty messages")
	}
}
