package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type DrainOptions struct {
	Namespace     string
	LabelSelector string
	Timeout       time.Duration
}

func DrainNode(ctx context.Context, client kubernetes.Interface, name string, opts DrainOptions) error {
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required for drain")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	pods, err := client.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", name),
		LabelSelector: opts.LabelSelector,
	})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if isMirrorPod(&pod) || isDaemonSetPod(&pod) {
			continue
		}
		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		}
		err := client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return wait.PollImmediate(2*time.Second, opts.Timeout, func() (bool, error) {
		remaining, err := client.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", name),
			LabelSelector: opts.LabelSelector,
		})
		if err != nil {
			return false, err
		}
		for _, pod := range remaining.Items {
			if isMirrorPod(&pod) || isDaemonSetPod(&pod) {
				continue
			}
			return false, nil
		}
		return true, nil
	})
}

func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func isMirrorPod(pod *corev1.Pod) bool {
	_, ok := pod.Annotations["kubernetes.io/config.mirror"]
	return ok
}
