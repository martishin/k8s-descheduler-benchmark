package benchmark

import (
	"context"
	"testing"

	"k8s-descheduler-benchmark/internal/workloads"
)

func TestNewMaintenanceRunnerDefaults(t *testing.T) {
	cfg := MaintenanceConfig{
		WorkloadMix: workloads.Mix{"small": 10},
	}
	runner := newMaintenanceRunner(context.Background(), nil, cfg)
	if runner.workloadName != "deschedbench" {
		t.Fatalf("unexpected workload name: %s", runner.workloadName)
	}
	if runner.totalPods != 10 {
		t.Fatalf("unexpected total pods: %d", runner.totalPods)
	}
}
