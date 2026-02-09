package descheduler

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureInstalledMissingFields(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	cases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "missing namespace",
			cfg: Config{
				Image:      "registry.k8s.io/descheduler/descheduler:v0.32.2",
				PolicyYAML: "apiVersion: v1\nkind: ConfigMap\n",
			},
		},
		{
			name: "missing policy",
			cfg: Config{
				Namespace: "deschedbench-test",
				Image:     "registry.k8s.io/descheduler/descheduler:v0.32.2",
			},
		},
	}

	for _, tc := range cases {
		if err := EnsureInstalled(ctx, client, tc.cfg); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestEnsureInstalledCreatesResources(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "deschedbench-test"}})

	policy := "apiVersion: descheduler/v1alpha2\nkind: DeschedulerPolicy\n"
	cfg := Config{
		Namespace:  "deschedbench-test",
		Image:      "registry.k8s.io/descheduler/descheduler:v0.32.2",
		PolicyYAML: policy,
	}

	if err := EnsureInstalled(ctx, client, cfg); err != nil {
		t.Fatalf("EnsureInstalled failed: %v", err)
	}

	if _, err := client.CoreV1().ServiceAccounts(cfg.Namespace).Get(ctx, "deschedbench-descheduler", metav1.GetOptions{}); err != nil {
		t.Fatalf("service account missing: %v", err)
	}
	if _, err := client.RbacV1().ClusterRoles().Get(ctx, "deschedbench-descheduler", metav1.GetOptions{}); err != nil {
		t.Fatalf("cluster role missing: %v", err)
	}
	binding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, "deschedbench-descheduler", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("cluster role binding missing: %v", err)
	}
	if len(binding.Subjects) == 0 || binding.Subjects[0].Namespace != cfg.Namespace {
		t.Fatalf("cluster role binding subject namespace mismatch")
	}
	cm, err := client.CoreV1().ConfigMaps(cfg.Namespace).Get(ctx, "deschedbench-policy", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("configmap missing: %v", err)
	}
	if cm.Data["policy.yaml"] != policy {
		t.Fatalf("policy mismatch: %q", cm.Data["policy.yaml"])
	}
	if _, err := client.CoreV1().Services(cfg.Namespace).Get(ctx, "deschedbench-descheduler-metrics", metav1.GetOptions{}); err != nil {
		t.Fatalf("metrics service missing: %v", err)
	}
}

func TestRunOnceCreatesJob(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "deschedbench-test"}})

	cfg := Config{
		Namespace:  "deschedbench-test",
		Image:      "registry.k8s.io/descheduler/descheduler:v0.32.2",
		PolicyYAML: "apiVersion: descheduler/v1alpha2\nkind: DeschedulerPolicy\n",
	}

	if err := RunOnce(ctx, client, cfg); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	job, err := client.BatchV1().Jobs(cfg.Namespace).Get(ctx, "deschedbench-descheduler", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("job missing: %v", err)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
	}
	if job.Spec.Template.Spec.Containers[0].Image != cfg.Image {
		t.Fatalf("job image mismatch: %s", job.Spec.Template.Spec.Containers[0].Image)
	}
}
