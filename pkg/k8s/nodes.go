package k8s

import (
	"context"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ListNodes(ctx context.Context, client kubernetes.Interface, selector string) ([]corev1.Node, error) {
	list, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func UnschedulableNodeNames(nodes []corev1.Node) []string {
	var names []string
	for _, node := range nodes {
		if node.Spec.Unschedulable {
			names = append(names, node.Name)
			continue
		}
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/unschedulable" && taint.Effect == corev1.TaintEffectNoSchedule {
				names = append(names, node.Name)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}
