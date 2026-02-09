package benchmark

import (
	"time"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/workloads"
)

type MaintenanceConfig struct {
	RunID             string
	Namespace         string
	WorkloadImage     string
	WorkloadMix       workloads.Mix
	SizeClasses       map[string]workloads.SizeClass
	LabelSelector     string
	Labels            map[string]string
	RecordPhase       func(name string) error
	WaitTimeout       time.Duration
	PostUncordonWait  time.Duration
	DrainIterations   int
	DeschedulerImage  string
	DeschedulerNS     string
	DeschedulerPolicy string
	DeschedulerCron   string
}

type ScenarioResult struct {
	Evictions []k8s.EvictionRecord
	Duration  time.Duration
	DrainNode string
}
