package metrics

import (
	"context"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

type Sampler struct {
	client   kubernetes.Interface
	interval time.Duration
	snapOpts SnapshotOptions

	mu        sync.Mutex
	snapshots []Snapshot
	samples   []Sample
}

func NewSampler(client kubernetes.Interface, interval time.Duration, snapOpts SnapshotOptions) *Sampler {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Sampler{
		client:   client,
		interval: interval,
		snapOpts: snapOpts,
	}
}

func (s *Sampler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		snapshot, err := CollectSnapshot(ctx, s.client, s.snapOpts)
		if err == nil {
			sample := DeriveSample(snapshot)
			s.mu.Lock()
			s.snapshots = append(s.snapshots, snapshot)
			s.samples = append(s.samples, sample)
			s.mu.Unlock()
			RecordSample(sample)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Sampler) Samples() []Sample {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Sample, len(s.samples))
	copy(out, s.samples)
	return out
}

func (s *Sampler) Snapshots() []Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Snapshot, len(s.snapshots))
	copy(out, s.snapshots)
	return out
}
