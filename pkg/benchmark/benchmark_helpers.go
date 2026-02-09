package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/report"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func countPodsOnNode(ctx context.Context, client kubernetes.Interface, namespace, labelSelector, nodeName string) (int, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
		LabelSelector: labelSelector,
	})
	if err != nil {
		return 0, err
	}
	return len(pods.Items), nil
}

func logSchedulingSummary(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string, logger *slog.Logger) {
	summary, err := k8s.SummarizeScheduling(ctx, client, namespace, labelSelector)
	if err != nil {
		return
	}
	logger.Info("scheduling summary",
		logging.StringField("ready", fmt.Sprintf("%d", summary.Ready)),
		logging.StringField("pending", fmt.Sprintf("%d", summary.Pending)),
		logging.StringField("reasons", report.FormatScheduleMessages(summary.Messages)),
	)
	nodes, err := k8s.ListNodes(ctx, client, "")
	if err != nil {
		return
	}
	var unsched []string
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			unsched = append(unsched, node.Name)
		}
	}
	if len(unsched) > 0 {
		logger.Info("unschedulable nodes", logging.StringField("nodes", strings.Join(unsched, ",")))
	}
}

func countSchedulableWorkers(nodes []corev1.Node) int {
	count := 0
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		if isControlPlaneNode(node.Labels) {
			continue
		}
		count++
	}
	return count
}

func pickDrainNode(nodes []corev1.Node) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes found")
	}
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		if isControlPlaneNode(node.Labels) {
			continue
		}
		return node.Name, nil
	}
	return nodes[0].Name, nil
}

func pickDrainNodeExcluding(nodes []corev1.Node, excluded map[string]struct{}) (string, error) {
	candidates := make([]corev1.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			continue
		}
		if isControlPlaneNode(node.Labels) {
			continue
		}
		if _, ok := excluded[node.Name]; ok {
			continue
		}
		candidates = append(candidates, node)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no available drain nodes found")
	}
	return candidates[0].Name, nil
}

func formatPhaseMessage(name string) string {
	switch name {
	case "workload:create":
		return "workload creation start"
	case "workload:ready":
		return "workload waiting done"
	case "snapshot:before":
		return "snapshot before start"
	case "snapshot:after":
		return "snapshot after benchmark start"
	default:
		return strings.ReplaceAll(name, ":", " ")
	}
}

func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return args
}

func isControlPlaneNode(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	if _, ok := labels["node-role.kubernetes.io/control-plane"]; ok {
		return true
	}
	if _, ok := labels["node-role.kubernetes.io/master"]; ok {
		return true
	}
	return false
}
