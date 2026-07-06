/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8sutil

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
	"github.com/gobackup/gobackup-operator/pkg/gobackupconfig"
)

// gobackupConfigKey is the Secret data key holding the rendered gobackup.yml.
const gobackupConfigKey = "gobackup.yml"

// CreateSecret renders the gobackup.yml for backup and stores it in a Secret
// (named after the Backup) owned by the Backup resource, creating or updating
// as needed.
func (k *K8s) CreateSecret(ctx context.Context, backup *backupv1.Backup) error {
	if backup == nil {
		return fmt.Errorf("backup cannot be nil")
	}
	namespace := backup.Namespace

	dbByName := make(map[string]*backupv1.Database)
	for _, ref := range backup.Spec.DatabaseRefs {
		db, err := k.getDatabase(ctx, namespace, ref.Name)
		if err != nil {
			return err
		}
		dbByName[ref.Name] = db
	}

	storageByName := make(map[string]*backupv1.Storage)
	for _, ref := range backup.Spec.StorageRefs {
		storage, err := k.getStorage(ctx, namespace, ref.Name)
		if err != nil {
			return err
		}
		storageByName[ref.Name] = storage
	}

	yamlData, err := gobackupconfig.Build(ctx, backup, dbByName, storageByName, k.resolveSecret)
	if err != nil {
		return fmt.Errorf("failed to build gobackup config: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			gobackupConfigKey: string(yamlData),
		},
	}

	ownerRef := metav1.NewControllerRef(backup, backupv1.GroupVersion.WithKind("Backup"))
	if ownerRef != nil {
		secret.OwnerReferences = append(secret.OwnerReferences, *ownerRef)
	}

	found, err := k.Clientset.CoreV1().Secrets(namespace).Get(ctx, backup.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err := k.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing secret: %w", err)
	}

	existing := found.DeepCopy()
	existing.StringData = secret.StringData
	if ownerRef != nil {
		existing.OwnerReferences = ensureOwnerReference(existing.OwnerReferences, *ownerRef)
	}
	if _, err := k.Clientset.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// getDatabase fetches a Database custom resource and decodes it into the typed
// struct, so config emission works from JSON tags rather than an unstructured map.
func (k *K8s) getDatabase(ctx context.Context, namespace, name string) (*backupv1.Database, error) {
	obj, err := k.GetCustomResource(ctx, backupv1.GroupVersion.Group, backupv1.GroupVersion.Version, backupv1.ResourceDatabases, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get database %s: %w", name, err)
	}
	var db backupv1.Database
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &db); err != nil {
		return nil, fmt.Errorf("failed to decode database %s: %w", name, err)
	}
	return &db, nil
}

// getStorage fetches a Storage custom resource and decodes it into the typed struct.
func (k *K8s) getStorage(ctx context.Context, namespace, name string) (*backupv1.Storage, error) {
	obj, err := k.GetCustomResource(ctx, backupv1.GroupVersion.Group, backupv1.GroupVersion.Version, backupv1.ResourceStorages, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage %s: %w", name, err)
	}
	var storage backupv1.Storage
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &storage); err != nil {
		return nil, fmt.Errorf("failed to decode storage %s: %w", name, err)
	}
	return &storage, nil
}

// resolveSecret is the gobackupconfig.SecretResolver backed by the Kubernetes
// API: it reads the named key from a Secret, falling back to StringData.
func (k *K8s) resolveSecret(ctx context.Context, namespace, name, key string) (string, error) {
	secret, err := k.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("secret %s not found", name)
		}
		return "", fmt.Errorf("failed to get secret %s: %w", name, err)
	}
	if value, ok := secret.Data[key]; ok {
		return string(value), nil
	}
	if value, ok := secret.StringData[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("key %s not found in secret %s", key, name)
}

// ensureOwnerReference appends owner if an owner with the same UID is not present.
func ensureOwnerReference(refs []metav1.OwnerReference, owner metav1.OwnerReference) []metav1.OwnerReference {
	for _, ref := range refs {
		if ref.UID == owner.UID {
			return refs
		}
	}
	return append(refs, owner)
}
