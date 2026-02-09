package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForPodsReady(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string, expected int32, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return false, err
		}
		var readyCount int32
		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			if isPodReady(&pod) {
				readyCount++
			}
		}
		return readyCount == expected, nil
	})
}

func ListPods(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) ([]corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

type ScheduleSummary struct {
	Ready    int32
	Pending  int32
	Messages map[string]int
}

func SummarizeScheduling(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) (ScheduleSummary, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return ScheduleSummary{}, err
	}

	summary := ScheduleSummary{Messages: map[string]int{}}
	for _, pod := range pods.Items {
		if isPodReady(&pod) {
			summary.Ready++
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Spec.NodeName == "" {
			summary.Pending++
			if reason, message := podSchedulingFailure(&pod); message != "" {
				key := reason
				if key == "" {
					key = "unspecified"
				}
				summary.Messages[fmt.Sprintf("%s: %s", key, message)]++
			}
		}
	}
	return summary, nil
}

func podSchedulingFailure(pod *corev1.Pod) (string, string) {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
			return cond.Reason, cond.Message
		}
	}
	return "", ""
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
