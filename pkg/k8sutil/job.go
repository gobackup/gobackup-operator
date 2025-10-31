package k8sutil

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteJob deletes a Kubernetes Job in the specified namespace
func (k *K8s) DeleteJob(ctx context.Context, namespace, name string) error {
	err := k.Clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Job already deleted, this is fine
			return nil
		}
		return fmt.Errorf("failed to delete job %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}
