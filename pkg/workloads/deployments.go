package workloads

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type SizeClass struct {
	Name   string
	CPU    string
	Memory string
}

type WorkloadConfig struct {
	Namespace      string
	NamePrefix     string
	Labels         map[string]string
	Mix            Mix
	SizeClasses    map[string]SizeClass
	PodImage       string
	PodAnnotations map[string]string
	PodLabels      map[string]string
}

func EnsureWorkloads(ctx context.Context, client kubernetes.Interface, cfg WorkloadConfig) error {
	for className, count := range cfg.Mix {
		if count == 0 {
			continue
		}
		size, ok := cfg.SizeClasses[className]
		if !ok {
			return fmt.Errorf("size class %q not defined", className)
		}
		name := fmt.Sprintf("%s-%s", cfg.NamePrefix, className)
		if err := ensureDeployment(ctx, client, cfg, name, size, count); err != nil {
			return err
		}
	}
	return nil
}

func WaitForWorkloadsReady(ctx context.Context, client kubernetes.Interface, namespace, namePrefix string, mix Mix, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		var desired int32
		var ready int32
		for className, count := range mix {
			if count == 0 {
				continue
			}
			desired += count
			name := fmt.Sprintf("%s-%s", namePrefix, className)
			dep, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			ready += dep.Status.ReadyReplicas
		}
		return ready == desired, nil
	})
}

func ensureDeployment(ctx context.Context, client kubernetes.Interface, cfg WorkloadConfig, name string, size SizeClass, replicas int32) error {
	labels := map[string]string{}
	for k, v := range cfg.Labels {
		labels[k] = v
	}
	labels["app.kubernetes.io/name"] = name

	podLabels := map[string]string{}
	for k, v := range labels {
		podLabels[k] = v
	}
	for k, v := range cfg.PodLabels {
		podLabels[k] = v
	}

	podAnnotations := map[string]string{}
	for k, v := range cfg.PodAnnotations {
		podAnnotations[k] = v
	}

	cpuQty := resource.MustParse(size.CPU)
	memQty := resource.MustParse(size.Memory)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: cfg.PodImage,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cpuQty,
									corev1.ResourceMemory: memQty,
								},
							},
						},
					},
				},
			},
		},
	}

	existing, err := client.AppsV1().Deployments(cfg.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil && existing != nil {
		dep.ResourceVersion = existing.ResourceVersion
		_, err = client.AppsV1().Deployments(cfg.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
		return err
	}
	_, err = client.AppsV1().Deployments(cfg.Namespace).Create(ctx, dep, metav1.CreateOptions{})
	return err
}
