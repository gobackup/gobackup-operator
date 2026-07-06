package k8sutil

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCustomResource fetches a namespaced custom-resource instance via the dynamic client.
func (k *K8s) GetCustomResource(ctx context.Context, group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	// Fetch the instance
	crdObj, err := k.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("custom resource %s in namespace %s not found: %w", name, namespace, err)
		}

		return nil, fmt.Errorf("failed to fetch custom resource %s in namespace %s: %w", name, namespace, err)
	}

	return crdObj, nil
}
