package cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scope struct {
	Namespace       string
	NamespacePrefix string
	Wait            bool
}

type CleanupService struct {
	client kubernetes.Interface
	logger *slog.Logger
}

func NewCleanupService(client kubernetes.Interface, logger *slog.Logger) *CleanupService {
	if logger == nil {
		logger = logging.GetLogger()
	}
	return &CleanupService{
		client: client,
		logger: logger,
	}
}

func (s *CleanupService) Preflight(ctx context.Context) error {
	nodes, err := k8s.ListNodes(ctx, s.client, "")
	if err != nil {
		return err
	}
	unschedulable := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if !node.Spec.Unschedulable {
			continue
		}
		if isControlPlaneNode(node.Labels) {
			continue
		}
		unschedulable = append(unschedulable, node.Name)
	}
	if len(unschedulable) > 0 {
		return fmt.Errorf("refusing to run: nodes unschedulable before benchmark: %s. Run `make cleanup`", strings.Join(unschedulable, ", "))
	}
	return nil
}

func (s *CleanupService) Run(ctx context.Context, scope Scope) error {
	if err := s.cleanupNamespaces(ctx, scope); err != nil {
		return err
	}
	if err := s.uncordonNodes(ctx); err != nil {
		return err
	}
	return nil
}

func (s *CleanupService) cleanupNamespaces(ctx context.Context, scope Scope) error {
	switch {
	case scope.Namespace != "":
		if err := k8s.DeleteNamespace(ctx, s.client, scope.Namespace); err != nil {
			return err
		}
		if scope.Wait {
			if err := k8s.WaitForNamespaceDeleted(ctx, s.client, scope.Namespace, 10*time.Minute); err != nil {
				return err
			}
		}
		return nil
	case scope.NamespacePrefix != "":
		return s.cleanupNamespacePrefix(ctx, scope.NamespacePrefix, scope.Wait)
	default:
		return s.cleanupNamespacePrefix(ctx, "deschedbench-", scope.Wait)
	}
}

func (s *CleanupService) cleanupNamespacePrefix(ctx context.Context, prefix string, wait bool) error {
	namespaces, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		if !strings.HasPrefix(ns.Name, prefix) {
			continue
		}
		if err := k8s.DeleteNamespace(ctx, s.client, ns.Name); err != nil {
			return err
		}
		if wait {
			if err := k8s.WaitForNamespaceDeleted(ctx, s.client, ns.Name, 10*time.Minute); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *CleanupService) uncordonNodes(ctx context.Context) error {
	nodes, err := k8s.ListNodes(ctx, s.client, "")
	if err != nil {
		return err
	}
	unschedulable := k8s.UnschedulableNodeNames(nodes)
	for _, name := range unschedulable {
		s.logger.Info("uncordon node", logging.StringField("name", name))
		if err := k8s.UncordonNode(ctx, s.client, name); err != nil {
			return err
		}
	}
	return nil
}

func isControlPlaneNode(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	if _, ok := labels["node-role.kubernetes.io/control-plane"]; ok {
		return true
	}
	if _, ok := labels["node-role.kubernetes.io/master"]; ok {
		return true
	}
	return false
}
