package benchmark

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"k8s-descheduler-benchmark/internal/benchmark"
	"k8s-descheduler-benchmark/internal/k8s"
	"k8s-descheduler-benchmark/internal/logging"
	"k8s-descheduler-benchmark/internal/metrics"
	"k8s-descheduler-benchmark/internal/report"
	"k8s-descheduler-benchmark/internal/service/cleanup"
	"k8s.io/client-go/kubernetes"
)

const (
	scenarioName             = "maintenance"
	deschedulerImagePinned   = "registry.k8s.io/descheduler/descheduler:v0.32.2"
	deschedulerCronPinned    = "job"
	defaultResultsDir        = "results"
	defaultSampleInterval    = 5 * time.Second
	defaultPostUncordonWait  = 60 * time.Second
	defaultWaitTimeout       = 10 * time.Minute
	defaultBalanceStddevGoal = 1.0
)

type Runner struct {
	Client      kubernetes.Interface
	Cleanup     *cleanup.CleanupService
	Logger      *slog.Logger
	MetricsPort int
}

type RunConfig struct {
	PodsTotal  int32
	PodCPU     string
	PodMemory  string
	Profile    string
	OutputPath string
	Context    string
	Server     string
}

func (r *Runner) Run(ctx context.Context, cfg RunConfig) error {
	if r.Client == nil {
		return fmt.Errorf("client is required")
	}
	logger := r.Logger
	if logger == nil {
		logger = logging.GetLogger()
	}
	cleanupSvc := r.Cleanup
	if cleanupSvc == nil {
		cleanupSvc = cleanup.NewCleanupService(r.Client, logger)
	}
	if err := cleanupSvc.Preflight(ctx); err != nil {
		return err
	}

	plan, err := NewPlanBuilder().Build(cfg)
	if err != nil {
		return err
	}

	ctxRun, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		sig := <-sigCh
		logger.Info("signal received", logging.StringField("signal", sig.String()))
		cancel()
	}()

	cleanupOnce := sync.Once{}
	runCleanup := func(reason string) {
		cleanupOnce.Do(func() {
			logger.Info("namespace cleanup start",
				logging.StringField("reason", reason),
				logging.StringField("namespace", plan.Namespace),
			)
			ctxCleanup, cancelCleanup := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancelCleanup()
			if err := cleanupSvc.Run(ctxCleanup, cleanup.Scope{
				Namespace: plan.Namespace,
				Wait:      true,
			}); err != nil {
				logger.Error("cleanup failed", logging.ErrorField(err))
				return
			}
			logger.Info("namespace cleanup done", logging.StringField("namespace", plan.Namespace))
		})
	}
	defer runCleanup("defer")

	logger.Info("benchmark namespace", logging.StringField("value", plan.Namespace))
	logger.Info("results file path", logging.StringField("value", plan.OutputPath))
	logger.Info("metrics server start", logging.StringField("port", fmt.Sprintf("%d", r.MetricsPort)))
	logger.Info("run id", logging.StringField("value", plan.RunID))

	metrics.StartMetricsServer(r.MetricsPort)

	metrics.RunInfo.WithLabelValues(scenarioName, cfg.Profile, plan.RunID).Set(1)
	defer metrics.RunInfo.WithLabelValues(scenarioName, cfg.Profile, plan.RunID).Set(0)

	sampler := metrics.NewSampler(r.Client, defaultSampleInterval, metrics.SnapshotOptions{
		Namespace:     plan.Namespace,
		NamespaceOnly: true,
	})
	go sampler.Run(ctxRun)

	phaseRec := NewPhaseRecorder(r.Client, metrics.SnapshotOptions{
		Namespace:     plan.Namespace,
		NamespaceOnly: true,
	}, logger)

	logger.Info("starting maintenance scenario")
	result, err := benchmark.RunMaintenance(ctxRun, r.Client, benchmark.MaintenanceConfig{
		RunID:         plan.RunID,
		Namespace:     plan.Namespace,
		WorkloadImage: "registry.k8s.io/pause:3.9",
		WorkloadMix:   plan.Mix,
		SizeClasses:   plan.SizeClasses,
		LabelSelector: plan.LabelSelector,
		Labels:        plan.Labels,
		RecordPhase: func(name string) error {
			return phaseRec.Record(ctxRun, name)
		},
		WaitTimeout:       defaultWaitTimeout,
		PostUncordonWait:  defaultPostUncordonWait,
		DrainIterations:   2,
		DeschedulerImage:  deschedulerImagePinned,
		DeschedulerNS:     plan.Namespace,
		DeschedulerPolicy: plan.PolicyYAML,
		DeschedulerCron:   deschedulerCronPinned,
	})
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("scenario").Inc()
		if errors.Is(err, context.Canceled) {
			runCleanup("cancel")
			return err
		}
		runCleanup("error")
		return err
	}

	cancel()
	samples := sampler.Samples()
	phases := phaseRec.Phases()
	beforeSnap, beforeSample := phaseRec.Before()
	afterSnap, afterSample := phaseRec.After()

	rebalanceTime := computeRebalanceTime(samples, phases, defaultBalanceStddevGoal)
	summary := report.Summary{
		RunID:                plan.RunID,
		Scenario:             scenarioName,
		Profile:              cfg.Profile,
		DurationSeconds:      result.Duration.Seconds(),
		RebalanceTimeSeconds: rebalanceTime,
		Before:               beforeSample,
		After:                afterSample,
	}

	config := report.RunConfig{
		RunID:                plan.RunID,
		Scenario:             scenarioName,
		Profile:              cfg.Profile,
		Namespace:            plan.Namespace,
		StartTime:            time.Now(),
		Context:              cfg.Context,
		Server:               cfg.Server,
		PodsTotal:            cfg.PodsTotal,
		PodCPU:               cfg.PodCPU,
		PodMemory:            cfg.PodMemory,
		DeschedulerImage:     deschedulerImagePinned,
		DeschedulerNamespace: plan.Namespace,
		DeschedulerCron:      deschedulerCronPinned,
		SampleInterval:       defaultSampleInterval.String(),
		SampleDuration:       "0s",
	}

	output := struct {
		Config         report.RunConfig     `json:"config"`
		Phases         []report.PhaseMarker `json:"phases"`
		Summary        report.Summary       `json:"summary"`
		Samples        []metrics.Sample     `json:"samples"`
		BeforeSnapshot metrics.Snapshot     `json:"before_snapshot"`
		AfterSnapshot  metrics.Snapshot     `json:"after_snapshot"`
		Evictions      []k8s.EvictionRecord `json:"evictions"`
	}{
		Config:         config,
		Phases:         phases,
		Summary:        summary,
		Samples:        samples,
		BeforeSnapshot: beforeSnap,
		AfterSnapshot:  afterSnap,
		Evictions:      result.Evictions,
	}

	if err := report.WriteJSON(plan.OutputPath, output); err != nil {
		runCleanup("error")
		return err
	}

	metrics.TotalDuration.WithLabelValues(scenarioName, cfg.Profile, plan.RunID).Set(result.Duration.Seconds())
	logSummary(summary, beforeSnap, afterSnap)
	logger.Info("benchmark completed")
	runCleanup("success")
	logger.Info("results output", logging.StringField("path", plan.OutputPath))
	return nil
}
