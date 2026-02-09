package report

import (
	"time"

	"k8s-descheduler-benchmark/internal/metrics"
)

type RunConfig struct {
	RunID                string    `json:"run_id"`
	Scenario             string    `json:"scenario"`
	Profile              string    `json:"profile"`
	Namespace            string    `json:"namespace"`
	StartTime            time.Time `json:"start_time"`
	Context              string    `json:"context"`
	Server               string    `json:"server"`
	PodsTotal            int32     `json:"pods_total"`
	PodCPU               string    `json:"pod_cpu"`
	PodMemory            string    `json:"pod_memory"`
	DeschedulerImage     string    `json:"descheduler_image"`
	DeschedulerNamespace string    `json:"descheduler_namespace"`
	DeschedulerCron      string    `json:"descheduler_cron"`
	SampleInterval       string    `json:"sample_interval"`
	SampleDuration       string    `json:"sample_duration"`
}

type PhaseMarker struct {
	Name string    `json:"name"`
	Time time.Time `json:"time"`
}

type Summary struct {
	RunID                string         `json:"run_id"`
	Scenario             string         `json:"scenario"`
	Profile              string         `json:"profile"`
	DurationSeconds      float64        `json:"duration_seconds"`
	RebalanceTimeSeconds float64        `json:"rebalance_time_seconds"`
	Before               metrics.Sample `json:"before"`
	After                metrics.Sample `json:"after"`
}
