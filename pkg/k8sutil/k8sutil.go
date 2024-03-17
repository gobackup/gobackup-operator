package k8sutil

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewClient() (*kubernetes.Clientset, error) {
	// Create config to use the ServiceAccount's token, CA cert, and API server address
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func NewDynamicClient() (*dynamic.DynamicClient, error) {
	// Create config to use the ServiceAccount's token, CA cert, and API server address
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}
