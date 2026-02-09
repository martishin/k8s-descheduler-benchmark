package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnschedulableNodeNames(t *testing.T) {
	nodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Spec: corev1.NodeSpec{Unschedulable: true}},
		{ObjectMeta: metav1.ObjectMeta{Name: "c"}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: "node.kubernetes.io/unschedulable", Effect: corev1.TaintEffectNoSchedule}}}},
	}
	got := UnschedulableNodeNames(nodes)
	if len(got) != 2 {
		t.Fatalf("expected 2 unschedulable nodes, got %d", len(got))
	}
	if got[0] != "b" || got[1] != "c" {
		t.Fatalf("unexpected nodes: %v", got)
	}
}
