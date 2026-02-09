package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/workloads"
	"k8s.io/client-go/kubernetes"
)

type maintenanceRunner struct {
	ctx            context.Context
	client         kubernetes.Interface
	cfg            MaintenanceConfig
	logger         *slog.Logger
	workloadName   string
	totalPods      int32
	drainNode      string
	preEvictLabels map[string]string
	drainedNodes   map[string]struct{}
	iteration      int
}

func RunMaintenance(ctx context.Context, client kubernetes.Interface, cfg MaintenanceConfig) (ScenarioResult, error) {
	start := time.Now()
	runner := newMaintenanceRunner(ctx, client, cfg)

	if err := runner.prepareWorkloads(); err != nil {
		return ScenarioResult{}, err
	}
	if err := runner.snapshotBefore(); err != nil {
		return ScenarioResult{}, err
	}
	if err := runner.ensureDescheduler(); err != nil {
		return ScenarioResult{}, err
	}
	if err := runner.capturePreEvictions(); err != nil {
		return ScenarioResult{}, err
	}

	iterations := cfg.DrainIterations
	if iterations <= 0 {
		iterations = 1
	}
	if err := runner.validateIterations(iterations); err != nil {
		return ScenarioResult{}, err
	}

	for i := 0; i < iterations; i++ {
		runner.iteration = i + 1
		if err := runner.mark("maintenance:iteration", logging.StringField("iteration", fmt.Sprintf("%d", runner.iteration))); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.selectDrainNode(); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.cordonAndDrain(); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.waitForReschedule(); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.uncordon(); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.runDescheduler(); err != nil {
			return ScenarioResult{}, err
		}
		if err := runner.waitPostUncordon(); err != nil {
			return ScenarioResult{}, err
		}
	}
	if err := runner.snapshotAfter(); err != nil {
		return ScenarioResult{}, err
	}

	evictions := runner.collectEvictions()

	return ScenarioResult{
		Evictions: evictions,
		Duration:  time.Since(start),
		DrainNode: runner.drainNode,
	}, nil
}

func newMaintenanceRunner(ctx context.Context, client kubernetes.Interface, cfg MaintenanceConfig) *maintenanceRunner {
	return &maintenanceRunner{
		ctx:          ctx,
		client:       client,
		cfg:          cfg,
		logger:       logging.GetLogger(),
		workloadName: "deschedbench",
		totalPods:    workloads.MixTotal(cfg.WorkloadMix),
		drainedNodes: map[string]struct{}{},
	}
}
