package metrics

func RecordSample(sample Sample) {
	PodsStddev.Set(sample.PodsStddev)
	PodsMaxMinRatio.Set(sample.PodsMaxMinRatio)
	UnschedulablePods.Set(float64(sample.UnschedulablePods))
}

func RecordPhase(name string) {
	PhaseTotal.WithLabelValues(name).Inc()
}
