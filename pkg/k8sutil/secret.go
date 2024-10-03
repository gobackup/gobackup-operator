package k8sutil

import (
	"context"
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	databasev1 "github.com/gobackup/gobackup-operator/api/database/v1"
	storagev1 "github.com/gobackup/gobackup-operator/api/storage/v1"
	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
)

// BackupConfig represents the configuration for backups
type BackupConfig struct {
	Models Models `yaml:"models,omitempty"`
}

// Models represents the different models for backup configuration
type Models struct {
	// TODO: change my_backup to users backup name
	MyBackup MyBackup `yaml:"my_backup,omitempty"`
}

// MyBackup represents the configuration for "my_backup" model
type MyBackup struct {
	Databases Databases `yaml:"databases"`
	Storages  Storages  `yaml:"storages"`
	backupv1.BackupModelSpecConfig
}

// Databases represents the database configurations
type Databases struct {
	Postgres databasev1.PostgreSQLSpecConfig `yaml:"postgres"`
}

// Storages represents the storage configurations
type Storages struct {
	S3 storagev1.S3SpecConfig `yaml:"s3"`
}

// CreateSecret creates secret from config.
func (k *K8s) CreateSecret(ctx context.Context, model backupv1.Model, namespace, name string) error {
	var postgreSQLSpec databasev1.PostgreSQLSpec
	var s3Spec storagev1.S3Spec

	for _, database := range model.DatabaseRefs {
		version := strings.ToLower(database.Type) + "s"

		databaseCRD, err := k.GetCRD(ctx, database.APIGroup, "v1", version, namespace, database.Name)
		if err != nil {
			return err
		}

		specMap, ok := databaseCRD.Object["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("spec is not a valid map[string]interface{}")
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(specMap, &postgreSQLSpec); err != nil {
			return err
		}

		postgreSQLSpec.Type = strings.ToLower(database.Type)
	}

	for _, storage := range model.StorageRefs {
		version := strings.ToLower(storage.Type) + "s"

		storageCRD, err := k.GetCRD(ctx, storage.APIGroup, "v1", version, namespace, storage.Name)
		if err != nil {
			return err
		}

		specMap, ok := storageCRD.Object["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("spec is not a valid map[string]interface{}")
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(specMap, &s3Spec); err != nil {
			return err
		}

		s3Spec.Type = strings.ToLower(storage.Type)
	}

	backupConfig := BackupConfig{
		Models: Models{
			MyBackup: MyBackup{
				Databases: Databases{
					databasev1.PostgreSQLSpecConfig(postgreSQLSpec),
				},
				Storages: Storages{
					storagev1.S3SpecConfig(s3Spec),
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(&backupConfig)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: map[string]string{
			"gobackup.yml": string(yamlData),
		},
	}

	// Create the Secret in the specified namespace
	_, err = k.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	return nil
}

// CreateSecret creates secret from config.
func (k *K8s) DeleteSecret(ctx context.Context, namespace, name string) error {
	// Create the Secret in the specified namespace
	err := k.Clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		panic(err.Error())
	}

	return nil
}
