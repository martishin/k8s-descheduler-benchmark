package benchmark

import (
	"strings"
	"testing"
	"time"
)

func TestLabelsToSelector(t *testing.T) {
	labels := map[string]string{"a": "1", "b": "2"}
	selector := labelsToSelector(labels)
	parts := strings.Split(selector, ",")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	for _, part := range parts {
		if !strings.Contains(part, "=") {
			t.Fatalf("expected key=value, got %s", part)
		}
	}
}

func TestPlanBuilderDefaults(t *testing.T) {
	builder := &PlanBuilder{
		Now: func() time.Time {
			return time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)
		},
	}
	cfg := RunConfig{
		PodsTotal:  10,
		PodCPU:     "100m",
		PodMemory:  "128Mi",
		Profile:    "baseline",
		OutputPath: "",
	}
	plan, err := builder.Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if plan.RunID != "20260209-000000" {
		t.Fatalf("unexpected runID: %s", plan.RunID)
	}
	if plan.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	if !strings.HasPrefix(plan.Namespace, "deschedbench-") {
		t.Fatalf("unexpected namespace: %s", plan.Namespace)
	}
}
