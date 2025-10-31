package k8sutil

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8s struct {
	Clientset     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
}

// getConfig attempts to get in-cluster config, falling back to kubeconfig for local development
func getConfig() (*rest.Config, error) {
	// Try in-cluster config first (when running in a pod)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig for local development
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}

func NewClient() (*kubernetes.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	return kubernetes.NewForConfig(config)
}

func NewDynamicClient() (*dynamic.DynamicClient, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	// Create a dynamicClient from the configuration
	return dynamic.NewForConfig(config)
}
