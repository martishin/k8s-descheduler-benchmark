package benchmark

import (
	"fmt"

	"k8s-descheduler-benchmark/internal/logging"
	"k8s-descheduler-benchmark/internal/metrics"
	"k8s-descheduler-benchmark/internal/report"
)

func logSummary(summary report.Summary, before metrics.Snapshot, after metrics.Snapshot) {
	logger := logging.GetLogger()
	logger.Info("run summary",
		logging.StringField("duration", fmt.Sprintf("%.1fs", summary.DurationSeconds)),
		logging.StringField("rebalance_time", fmt.Sprintf("%.1fs", summary.RebalanceTimeSeconds)),
		logging.StringField("before_pods_stddev", fmt.Sprintf("%.3f", summary.Before.PodsStddev)),
		logging.StringField("after_pods_stddev", fmt.Sprintf("%.3f", summary.After.PodsStddev)),
		logging.StringField("before_pods", report.FormatNodePods(before)),
		logging.StringField("after_pods", report.FormatNodePods(after)),
	)
}
