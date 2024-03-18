package k8sutil

import (
	"context"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
)

// BackupConfig represents the configuration for backups
type BackupConfig struct {
	Models Models `yaml:"models,omitempty"`
}

// Models represents the different models for backup configuration
type Models struct {
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
	Postgres backupv1.PostgreSQLSpecConfig `yaml:"postgres"`
}

// Storages represents the storage configurations
type Storages struct {
	S3 backupv1.S3SpecConfig `yaml:"s3"`
}

// CreateSecret creates secret from config.
func CreateSecret(ctx context.Context, model backupv1.Model, clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, namespace string) error {
	var postgreSQLSpec backupv1.PostgreSQLSpec
	var s3Spec backupv1.S3Spec

	for _, database := range model.DatabaseRefs {
		databaseCRD, err := GetCRD(ctx, dynamicClient, "database.gobackup.io", "v1", "postgresqls", namespace, database.Name)
		if err != nil {
			return err
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(databaseCRD.Object["spec"].(map[string]interface{}), &postgreSQLSpec); err != nil {
			return err
		}

		postgreSQLSpec.Type = "postgresql"
	}

	for _, storage := range model.StorageRefs {
		storageCRD, err := GetCRD(ctx, dynamicClient, "storage.gobackup.io", "v1", "s3s", namespace, storage.Name)
		if err != nil {
			return err
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(storageCRD.Object["spec"].(map[string]interface{}), &s3Spec); err != nil {
			return err
		}

		s3Spec.Type = "s3"
	}

	backupConfig := BackupConfig{
		Models: Models{
			MyBackup: MyBackup{
				Databases: Databases{
					backupv1.PostgreSQLSpecConfig(postgreSQLSpec),
				},
				Storages: Storages{
					backupv1.S3SpecConfig(s3Spec),
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
			//TODO: Change name of secret to something unique
			Name: "gobackup-secret",
		},
		StringData: map[string]string{
			"gobackup.yml": string(yamlData),
		},
	}

	// Create the Secret in the specified namespace
	_, err = clientset.CoreV1().Secrets("gobackup-operator-test").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	return nil
}
