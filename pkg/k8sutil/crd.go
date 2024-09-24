package k8sutil

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GetCRD fetches a CRD instance.
func GetCRD(ctx context.Context, dynamicClient dynamic.Interface,
	group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	// Fetch the instance
	crdObj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("CRD %s in namespace %s not found: %w", name, namespace, err)
		}

		return nil, fmt.Errorf("failed to fetch CRD %s in namespace %s: %w", name, namespace, err)
	}

	return crdObj, nil
}
