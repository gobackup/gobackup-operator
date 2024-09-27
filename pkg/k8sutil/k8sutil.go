package k8sutil

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8s struct {
	Clientset     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
}

func NewClient() (*kubernetes.Clientset, error) {
	// Create config to use the ServiceAccount's token, CA cert, and API server address
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	return kubernetes.NewForConfig(config)
}

func NewDynamicClient() (*dynamic.DynamicClient, error) {
	// Create config to use the ServiceAccount's token, CA cert, and API server address
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create a dynamicClient from the configuration
	return dynamic.NewForConfig(config)
}
