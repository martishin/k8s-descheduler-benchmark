package workloads

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureWorkloadsCreatesDeployment(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}})

	cfg := WorkloadConfig{
		Namespace:  "test",
		NamePrefix: "bench",
		Labels: map[string]string{
			"deschedbench": "true",
		},
		Mix: Mix{"small": 3},
		SizeClasses: map[string]SizeClass{
			"small": {Name: "small", CPU: "100m", Memory: "128Mi"},
		},
		PodImage: "registry.k8s.io/pause:3.9",
		PodLabels: map[string]string{
			"tier": "test",
		},
		PodAnnotations: map[string]string{
			"note": "hello",
		},
	}

	if err := EnsureWorkloads(ctx, client, cfg); err != nil {
		t.Fatalf("EnsureWorkloads failed: %v", err)
	}

	dep, err := client.AppsV1().Deployments("test").Get(ctx, "bench-small", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected deployment created: %v", err)
	}

	if dep.Labels["deschedbench"] != "true" {
		t.Fatalf("expected base label on deployment")
	}
	if dep.Labels["app.kubernetes.io/name"] != "bench-small" {
		t.Fatalf("expected app label on deployment")
	}
	if dep.Spec.Template.Labels["tier"] != "test" {
		t.Fatalf("expected pod label on template")
	}
	if dep.Spec.Template.Annotations["note"] != "hello" {
		t.Fatalf("expected pod annotation on template")
	}

	reqs := dep.Spec.Template.Spec.Containers[0].Resources.Requests
	if reqs.Cpu().Cmp(resource.MustParse("100m")) != 0 {
		t.Fatalf("unexpected cpu request: %s", reqs.Cpu().String())
	}
	if reqs.Memory().Cmp(resource.MustParse("128Mi")) != 0 {
		t.Fatalf("unexpected memory request: %s", reqs.Memory().String())
	}
}

func TestEnsureWorkloadsMissingSizeClass(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	cfg := WorkloadConfig{
		Namespace:  "test",
		NamePrefix: "bench",
		Mix:        Mix{"small": 1},
		SizeClasses: map[string]SizeClass{
			"large": {Name: "large", CPU: "500m", Memory: "512Mi"},
		},
		PodImage: "registry.k8s.io/pause:3.9",
	}

	if err := EnsureWorkloads(ctx, client, cfg); err == nil {
		t.Fatalf("expected error for missing size class")
	}
}

func TestEnsureWorkloadsUpdatesExistingDeployment(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	existing := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bench-small",
			Namespace: "test",
			Labels:    map[string]string{"app.kubernetes.io/name": "bench-small"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "bench-small"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "bench-small"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: "registry.k8s.io/pause:3.9",
						},
					},
				},
			},
		},
	}

	if _, err := client.AppsV1().Deployments("test").Create(ctx, existing, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create existing deployment: %v", err)
	}

	cfg := WorkloadConfig{
		Namespace:  "test",
		NamePrefix: "bench",
		Labels:     map[string]string{"deschedbench": "true"},
		Mix:        Mix{"small": 4},
		SizeClasses: map[string]SizeClass{
			"small": {Name: "small", CPU: "200m", Memory: "256Mi"},
		},
		PodImage: "registry.k8s.io/pause:3.9",
	}

	if err := EnsureWorkloads(ctx, client, cfg); err != nil {
		t.Fatalf("EnsureWorkloads failed: %v", err)
	}

	updated, err := client.AppsV1().Deployments("test").Get(ctx, "bench-small", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected deployment updated: %v", err)
	}
	if updated.Spec.Replicas == nil || *updated.Spec.Replicas != 4 {
		t.Fatalf("expected replicas updated to 4")
	}
}

func TestWaitForWorkloadsReady(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bench-small",
			Namespace: "test",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "bench-small"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "bench-small"},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 2,
		},
	}
	if _, err := client.AppsV1().Deployments("test").Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	mix := Mix{"small": 2}
	if err := WaitForWorkloadsReady(ctx, client, "test", "bench", mix, 2*time.Second); err != nil {
		t.Fatalf("WaitForWorkloadsReady failed: %v", err)
	}
}

func int32Ptr(val int32) *int32 {
	return &val
}
