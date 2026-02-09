package benchmark

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/metrics"
	"k8s-descheduler-benchmark/pkg/report"
	"k8s.io/client-go/kubernetes"
)

type PhaseRecorder struct {
	client kubernetes.Interface
	opts   metrics.SnapshotOptions
	logger *slog.Logger

	mu           sync.Mutex
	phases       []report.PhaseMarker
	beforeSnap   metrics.Snapshot
	afterSnap    metrics.Snapshot
	beforeSample metrics.Sample
	afterSample  metrics.Sample
}

func NewPhaseRecorder(client kubernetes.Interface, opts metrics.SnapshotOptions, logger *slog.Logger) *PhaseRecorder {
	if logger == nil {
		logger = logging.GetLogger()
	}
	return &PhaseRecorder{
		client: client,
		opts:   opts,
		logger: logger,
	}
}

func (p *PhaseRecorder) Record(ctx context.Context, name string) error {
	now := time.Now()
	p.mu.Lock()
	p.phases = append(p.phases, report.PhaseMarker{Name: name, Time: now})
	p.mu.Unlock()
	metrics.RecordPhase(name)

	if name != "snapshot:before" && name != "snapshot:after" {
		return nil
	}

	snap, err := metrics.CollectSnapshot(ctx, p.client, p.opts)
	if err != nil {
		return err
	}
	sample := metrics.DeriveSample(snap)

	p.mu.Lock()
	if name == "snapshot:before" {
		p.beforeSnap = snap
		p.beforeSample = sample
	} else {
		p.afterSnap = snap
		p.afterSample = sample
	}
	p.mu.Unlock()

	p.logger.Info(snapshotDoneMessage(name),
		logging.StringField("metric", "pods per node (count)"),
		logging.StringField("pods_per_node", report.FormatNodePods(snap)),
	)

	return nil
}

func (p *PhaseRecorder) Phases() []report.PhaseMarker {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]report.PhaseMarker, len(p.phases))
	copy(out, p.phases)
	return out
}

func (p *PhaseRecorder) Before() (metrics.Snapshot, metrics.Sample) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.beforeSnap, p.beforeSample
}

func (p *PhaseRecorder) After() (metrics.Snapshot, metrics.Sample) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.afterSnap, p.afterSample
}

func snapshotDoneMessage(name string) string {
	switch name {
	case "snapshot:before":
		return "snapshot before done"
	case "snapshot:after":
		return "snapshot after benchmark done"
	default:
		return strings.ReplaceAll(name, "snapshot:", "snapshot ")
	}
}
