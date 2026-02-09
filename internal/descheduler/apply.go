package descheduler

import (
	"context"
	"fmt"
	"io"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func applyManifest(ctx context.Context, client kubernetes.Interface, manifest string) error {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	for {
		var raw runtime.RawExtension
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(raw.Raw) == 0 {
			continue
		}
		obj, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(raw.Raw, nil, nil)
		if err != nil {
			return err
		}
		if err := applyObject(ctx, client, obj); err != nil {
			return fmt.Errorf("apply %s failed: %w", gvk.String(), err)
		}
	}
}

func applyObject(ctx context.Context, client kubernetes.Interface, obj runtime.Object) error {
	switch typed := obj.(type) {
	case *corev1.ServiceAccount:
		return upsertServiceAccount(ctx, client, typed)
	case *rbacv1.ClusterRole:
		return upsertClusterRole(ctx, client, typed)
	case *rbacv1.ClusterRoleBinding:
		return upsertClusterRoleBinding(ctx, client, typed)
	case *corev1.ConfigMap:
		return upsertConfigMap(ctx, client, typed)
	case *corev1.Service:
		return upsertService(ctx, client, typed)
	case *batchv1.Job:
		return upsertJob(ctx, client, typed)
	default:
		return fmt.Errorf("unsupported object type %T", obj)
	}
}

func upsertServiceAccount(ctx context.Context, client kubernetes.Interface, sa *corev1.ServiceAccount) error {
	current, err := client.CoreV1().ServiceAccounts(sa.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		sa.ResourceVersion = current.ResourceVersion
		_, err = client.CoreV1().ServiceAccounts(sa.Namespace).Update(ctx, sa, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.CoreV1().ServiceAccounts(sa.Namespace).Create(ctx, sa, metav1.CreateOptions{})
	return err
}

func upsertClusterRole(ctx context.Context, client kubernetes.Interface, role *rbacv1.ClusterRole) error {
	current, err := client.RbacV1().ClusterRoles().Get(ctx, role.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		role.ResourceVersion = current.ResourceVersion
		_, err = client.RbacV1().ClusterRoles().Update(ctx, role, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
	return err
}

func upsertClusterRoleBinding(ctx context.Context, client kubernetes.Interface, binding *rbacv1.ClusterRoleBinding) error {
	current, err := client.RbacV1().ClusterRoleBindings().Get(ctx, binding.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		binding.ResourceVersion = current.ResourceVersion
		_, err = client.RbacV1().ClusterRoleBindings().Update(ctx, binding, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.RbacV1().ClusterRoleBindings().Create(ctx, binding, metav1.CreateOptions{})
	return err
}

func upsertConfigMap(ctx context.Context, client kubernetes.Interface, cm *corev1.ConfigMap) error {
	current, err := client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		cm.ResourceVersion = current.ResourceVersion
		_, err = client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func upsertService(ctx context.Context, client kubernetes.Interface, svc *corev1.Service) error {
	current, err := client.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		svc.ResourceVersion = current.ResourceVersion
		svc.Spec.ClusterIP = current.Spec.ClusterIP
		svc.Spec.ClusterIPs = current.Spec.ClusterIPs
		_, err = client.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	return err
}

func upsertJob(ctx context.Context, client kubernetes.Interface, job *batchv1.Job) error {
	current, err := client.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
	if err == nil && current != nil {
		job.ResourceVersion = current.ResourceVersion
		_, err = client.BatchV1().Jobs(job.Namespace).Update(ctx, job, metav1.UpdateOptions{})
		return err
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	_, err = client.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}
