package metrics

import (
	"math"
	"time"
)

type Sample struct {
	Time              time.Time `json:"time"`
	PodsStddev        float64   `json:"pods_stddev"`
	PodsMaxMinRatio   float64   `json:"pods_max_min_ratio"`
	UnschedulablePods int       `json:"unschedulable_pods"`
	NodesCount        int       `json:"nodes_count"`
	PodsCounted       int       `json:"pods_counted"`
}

func DeriveSample(snapshot Snapshot) Sample {
	podsPerNode := make([]float64, 0, len(snapshot.Nodes))

	for _, node := range snapshot.Nodes {
		podsPerNode = append(podsPerNode, float64(node.Pods))
	}

	podsStddev := stddev(podsPerNode)
	podsRatio := maxMinRatio(podsPerNode)

	return Sample{
		Time:              snapshot.Time,
		PodsStddev:        podsStddev,
		PodsMaxMinRatio:   podsRatio,
		UnschedulablePods: snapshot.UnschedulablePods,
		NodesCount:        len(snapshot.Nodes),
		PodsCounted:       snapshot.TotalPodsCounted,
	}
}

func stddev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	var sum float64
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	variance := sum / float64(len(values))
	return math.Sqrt(variance)
}

func maxMinRatio(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min_ := values[0]
	max_ := values[0]
	for _, v := range values[1:] {
		if v < min_ {
			min_ = v
		}
		if v > max_ {
			max_ = v
		}
	}
	if min_ == 0 {
		if max_ == 0 {
			return 0
		}
		return -1
	}
	return max_ / min_
}
