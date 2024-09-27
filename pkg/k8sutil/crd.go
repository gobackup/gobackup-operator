package k8sutil

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCRD fetches a CRD instance.
func (k *K8s) GetCRD(ctx context.Context, group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	// Fetch the instance
	crdObj, err := k.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("CRD %s in namespace %s not found: %w", name, namespace, err)
		}

		return nil, fmt.Errorf("failed to fetch CRD %s in namespace %s: %w", name, namespace, err)
	}

	return crdObj, nil
}
