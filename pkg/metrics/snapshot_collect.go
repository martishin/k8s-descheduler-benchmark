package metrics

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Snapshot struct {
	Time              time.Time            `json:"time"`
	Nodes             map[string]NodeStats `json:"nodes"`
	UnschedulablePods int                  `json:"unschedulable_pods"`
	TotalPodsCounted  int                  `json:"total_pods_counted"`
	Namespace         string               `json:"namespace"`
	NamespaceOnly     bool                 `json:"namespace_only"`
}

type NodeStats struct {
	Pods                int   `json:"pods"`
	CPURequestedMilli   int64 `json:"cpu_requested_milli"`
	MemRequestedBytes   int64 `json:"mem_requested_bytes"`
	CPUAllocatableMilli int64 `json:"cpu_allocatable_milli"`
	MemAllocatableBytes int64 `json:"mem_allocatable_bytes"`
}

type SnapshotOptions struct {
	Namespace     string
	NamespaceOnly bool
}

func CollectSnapshot(ctx context.Context, client kubernetes.Interface, opts SnapshotOptions) (Snapshot, error) {
	snap := Snapshot{
		Time:          time.Now(),
		Nodes:         map[string]NodeStats{},
		Namespace:     opts.Namespace,
		NamespaceOnly: opts.NamespaceOnly,
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return snap, err
	}
	for _, node := range nodes.Items {
		cpuAlloc := node.Status.Allocatable.Cpu().MilliValue()
		memAlloc := node.Status.Allocatable.Memory().Value()
		snap.Nodes[node.Name] = NodeStats{
			CPUAllocatableMilli: cpuAlloc,
			MemAllocatableBytes: memAlloc,
		}
	}

	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return snap, err
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		if opts.NamespaceOnly && opts.Namespace != "" && pod.Namespace != opts.Namespace {
			continue
		}
		if pod.Spec.NodeName != "" {
			stats := snap.Nodes[pod.Spec.NodeName]
			stats.Pods++
			cpuReq, memReq := podRequests(&pod)
			stats.CPURequestedMilli += cpuReq
			stats.MemRequestedBytes += memReq
			snap.Nodes[pod.Spec.NodeName] = stats
			snap.TotalPodsCounted++
		}

		if pod.Namespace == opts.Namespace {
			if isUnschedulable(&pod) {
				snap.UnschedulablePods++
			}
		}
	}

	return snap, nil
}

func podRequests(pod *corev1.Pod) (int64, int64) {
	var cpuMilli int64
	var memBytes int64
	for _, container := range pod.Spec.Containers {
		cpu := container.Resources.Requests.Cpu()
		mem := container.Resources.Requests.Memory()
		if cpu != nil {
			cpuMilli += cpu.MilliValue()
		}
		if mem != nil {
			memBytes += mem.Value()
		}
	}
	return cpuMilli, memBytes
}

func isUnschedulable(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse && cond.Reason == "Unschedulable" {
			return true
		}
	}
	return false
}
