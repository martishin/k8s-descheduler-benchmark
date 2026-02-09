package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientInfo struct {
	Context string `json:"context"`
	Server  string `json:"server"`
}

func NewClient(qps float32, burst int) (*kubernetes.Clientset, ClientInfo, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, ClientInfo{}, fmt.Errorf("failed to get raw kubeconfig: %v", err)
	}

	currentContext := rawConfig.CurrentContext
	if currentContext != "deschedbench" {
		return nil, ClientInfo{}, fmt.Errorf("refusing to run: kubeconfig context is %q, expected \"deschedbench\"", currentContext)
	}

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, ClientInfo{}, fmt.Errorf("failed to get REST config: %v", err)
	}

	restConfig.QPS = qps
	restConfig.Burst = burst

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, ClientInfo{}, fmt.Errorf("failed to create clientset: %v", err)
	}

	info := ClientInfo{
		Context: currentContext,
		Server:  restConfig.Host,
	}

	return clientset, info, nil
}
