package benchmark

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPickDrainNodeSkipsControlPlaneAndUnschedulable(t *testing.T) {
	nodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "cp", Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "unsched"}, Spec: corev1.NodeSpec{Unschedulable: true}},
		{ObjectMeta: metav1.ObjectMeta{Name: "worker"}},
	}
	got, err := pickDrainNode(nodes)
	if err != nil {
		t.Fatalf("pickDrainNode failed: %v", err)
	}
	if got != "worker" {
		t.Fatalf("expected worker, got %s", got)
	}
}

func TestFormatPhaseMessage(t *testing.T) {
	if got := formatPhaseMessage("snapshot:before"); got != "snapshot before start" {
		t.Fatalf("unexpected message: %s", got)
	}
	if got := formatPhaseMessage("snapshot:after"); got != "snapshot after benchmark start" {
		t.Fatalf("unexpected message: %s", got)
	}
	if got := formatPhaseMessage("cordon:start"); got != "cordon start" {
		t.Fatalf("unexpected message: %s", got)
	}
}

func TestAttrsToArgs(t *testing.T) {
	args := attrsToArgs(nil)
	if len(args) != 0 {
		t.Fatalf("expected empty args, got %d", len(args))
	}
}
