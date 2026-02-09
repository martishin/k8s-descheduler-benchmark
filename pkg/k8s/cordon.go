package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CordonNode(ctx context.Context, client kubernetes.Interface, name string) error {
	return updateNodeSchedulable(ctx, client, name, true)
}

func UncordonNode(ctx context.Context, client kubernetes.Interface, name string) error {
	return updateNodeSchedulable(ctx, client, name, false)
}

func updateNodeSchedulable(ctx context.Context, client kubernetes.Interface, name string, unschedulable bool) error {
	node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if node.Spec.Unschedulable == unschedulable {
		return nil
	}
	node.Spec.Unschedulable = unschedulable
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}
