package k8sutil

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GetBackup gets the backup CRD.
func GetBackup(ctx context.Context, clientset *kubernetes.Clientset,
	dynamicClient *dynamic.DynamicClient, namespace string) error {

	// Define the GVR (GroupVersionResource) for the backup CRD.
	backupGVR := schema.GroupVersionResource{
		Group:    "gobackup.io",
		Version:  "v1",
		Resource: "backups",
	}

	// Get the CRD in the specified namespace
	_, err := dynamicClient.Resource(backupGVR).Namespace(namespace).Get(ctx, "backup", metav1.GetOptions{})
	if err != nil {
		return err
	}

	return nil
}
