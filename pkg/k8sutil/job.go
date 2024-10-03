package k8sutil

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *K8s) DeleteJob(ctx context.Context, namespace, name string) error {
	err := k.Clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		panic(err.Error())
	}

	return nil
}
