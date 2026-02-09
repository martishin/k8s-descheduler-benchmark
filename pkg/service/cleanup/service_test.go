package cleanup

import (
	"context"
	"testing"

	"k8s-descheduler-benchmark/pkg/logging"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPreflightFailsWhenUnschedulableNode(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
	)
	service := NewCleanupService(client, logging.GetLogger())
	if err := service.Preflight(context.Background()); err == nil {
		t.Fatalf("expected error for unschedulable node")
	}
}

func TestPreflightPassesWhenSchedulable(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		},
	)
	service := NewCleanupService(client, logging.GetLogger())
	if err := service.Preflight(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRunCleansSingleNamespace(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "deschedbench-a"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "deschedbench-b"}},
	)
	service := NewCleanupService(client, logging.GetLogger())
	if err := service.Run(context.Background(), Scope{Namespace: "deschedbench-a", Wait: false}); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "deschedbench-a", metav1.GetOptions{}); err == nil {
		t.Fatalf("expected namespace deschedbench-a deleted")
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "deschedbench-b", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected namespace deschedbench-b to remain, got %v", err)
	}
}

func TestRunCleansByPrefix(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "deschedbench-a"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other"}},
	)
	service := NewCleanupService(client, logging.GetLogger())
	if err := service.Run(context.Background(), Scope{NamespacePrefix: "deschedbench-", Wait: false}); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "deschedbench-a", metav1.GetOptions{}); err == nil {
		t.Fatalf("expected namespace deschedbench-a deleted")
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "other", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected namespace other to remain, got %v", err)
	}
}

func TestRunUncordonsNodes(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
	)
	service := NewCleanupService(client, logging.GetLogger())
	if err := service.Run(context.Background(), Scope{NamespacePrefix: "deschedbench-", Wait: false}); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	node, err := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if node.Spec.Unschedulable {
		t.Fatalf("expected node to be uncordoned")
	}
}
