package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RunInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "deschedbench_run_info",
		Help: "1 if a benchmark run is currently active",
	}, []string{"scenario", "profile", "run_id"})

	TotalDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "deschedbench_total_duration_seconds",
		Help: "Total duration of the last benchmark run",
	}, []string{"scenario", "profile", "run_id"})

	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "deschedbench_errors_total",
		Help: "Total number of errors during benchmark",
	}, []string{"type"})

	PhaseTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "deschedbench_phase_total",
		Help: "Count of phase markers emitted",
	}, []string{"phase"})

	PodsStddev = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "deschedbench_pods_stddev",
		Help: "Pods per node standard deviation",
	})

	PodsMaxMinRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "deschedbench_pods_max_min_ratio",
		Help: "Pods per node max/min ratio",
	})

	UnschedulablePods = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "deschedbench_unschedulable_pods",
		Help: "Count of unschedulable pods in the benchmark namespace",
	})
)

var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(RunInfo)
	Registry.MustRegister(TotalDuration)
	Registry.MustRegister(ErrorsTotal)
	Registry.MustRegister(PhaseTotal)
	Registry.MustRegister(PodsStddev)
	Registry.MustRegister(PodsMaxMinRatio)
	Registry.MustRegister(UnschedulablePods)
}
