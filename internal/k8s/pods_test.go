package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSummarizeScheduling(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ready",
				Namespace: "ns",
				Labels:    map[string]string{"app": "x"},
			},
			Status: corev1.PodStatus{
				Phase:      corev1.PodRunning,
				Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pending",
				Namespace: "ns",
				Labels:    map[string]string{"app": "x"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				Conditions: []corev1.PodCondition{{
					Type:    corev1.PodScheduled,
					Status:  corev1.ConditionFalse,
					Reason:  "Unschedulable",
					Message: "0/1 nodes are available",
				}},
			},
		},
	)

	summary, err := SummarizeScheduling(context.Background(), client, "ns", "app=x")
	if err != nil {
		t.Fatalf("SummarizeScheduling failed: %v", err)
	}
	if summary.Ready != 1 {
		t.Fatalf("expected Ready=1, got %d", summary.Ready)
	}
	if summary.Pending != 1 {
		t.Fatalf("expected Pending=1, got %d", summary.Pending)
	}
	if len(summary.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(summary.Messages))
	}
}
