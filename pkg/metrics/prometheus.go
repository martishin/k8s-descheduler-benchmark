package metrics

import (
	"fmt"
	"net/http"

	"k8s-descheduler-benchmark/pkg/logging"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(port int) {
	http.Handle("/metrics", promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}))
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			logging.GetLogger().Error("metrics server error", logging.ErrorField(err))
		}
	}()
}
