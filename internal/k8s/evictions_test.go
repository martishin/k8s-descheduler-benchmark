package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodReadyTime(t *testing.T) {
	now := time.Now()
	pod := corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(now),
				},
			},
		},
	}
	if got := podReadyTime(&pod); got.IsZero() {
		t.Fatal("expected non-zero ready time")
	}
}

func TestEventTimestamp(t *testing.T) {
	now := time.Now()
	event := corev1.Event{
		EventTime: metav1.MicroTime{Time: now},
	}
	if got := eventTimestamp(&event); !got.Equal(now) {
		t.Fatalf("expected %v, got %v", now, got)
	}
}
