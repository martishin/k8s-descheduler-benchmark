package k8s

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func WaitForNamespaceDeleted(ctx context.Context, client kubernetes.Interface, name string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		_, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
