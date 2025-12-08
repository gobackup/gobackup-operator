package k8sutil

import (
	"context"
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
)

// BackupConfig represents the configuration for backups
type BackupConfig struct {
	Models map[string]Model `yaml:"models"`
}

// Model represents the configuration for a backup model
type Model struct {
	Databases map[string]interface{} `yaml:"databases"`
	Storages  map[string]interface{} `yaml:"storages"`

	// Optional fields
	BeforeScript string `yaml:"before_script,omitempty"`
	AfterScript  string `yaml:"after_script,omitempty"`

	// Compression and encryption
	Compress string `yaml:"compress_with,omitempty"`
	Encode   string `yaml:"encode_with,omitempty"`
}

// CreateSecret creates a secret containing the gobackup.yml configuration file
func (k *K8s) CreateSecret(ctx context.Context, model backupv1.BackupSpec, namespace, name string) error {
	databases := make(map[string]interface{})
	storages := make(map[string]interface{})

	// Process database references
	for _, database := range model.DatabaseRefs {
		dbType := strings.ToLower(database.Type)
		version := dbType + "s"

		// Fetch the database CRD
		databaseCRD, err := k.GetCRD(ctx, database.APIGroup, "v1", version, namespace, database.Name)
		if err != nil {
			return fmt.Errorf("failed to get %s database: %w", dbType, err)
		}

		// Extract the database spec
		specMap, ok := databaseCRD.Object["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("database spec for %s is not a valid map", database.Name)
		}

		// Convert field names to the format expected by gobackup (snake_case)
		dbConfig := make(map[string]interface{})
		for key, value := range specMap {
			// Convert camelCase to snake_case for applicable fields
			switch key {
			case "excludeTables":
				dbConfig["exclude_tables"] = value
			case "additionalOptions":
				dbConfig["additional_options"] = value
			default:
				dbConfig[key] = value
			}
		}

		// Set the database type explicitly
		dbConfig["type"] = dbType

		// Add to databases map
		databases[database.Name] = dbConfig
	}

	// Process storage references
	for _, storage := range model.StorageRefs {
		storageType := strings.ToLower(storage.Type)
		version := storageType + "s"

		// Fetch the storage CRD
		storageCRD, err := k.GetCRD(ctx, storage.APIGroup, "v1", version, namespace, storage.Name)
		if err != nil {
			return fmt.Errorf("failed to get %s storage: %w", storageType, err)
		}

		// Extract the storage spec
		specMap, ok := storageCRD.Object["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("storage spec for %s is not a valid map", storage.Name)
		}

		// Convert field names to the format expected by gobackup (snake_case)
		storageConfig := make(map[string]interface{})
		for key, value := range specMap {
			// Convert camelCase to snake_case for applicable fields
			switch key {
			case "accessKeyID":
				storageConfig["access_key_id"] = value
			case "secretAccessKey":
				storageConfig["secret_access_key"] = value
			case "forcePathStyle":
				storageConfig["force_path_style"] = value
			case "storageClass":
				storageConfig["storage_class"] = value
			case "maxRetries":
				storageConfig["max_retries"] = value
			default:
				storageConfig[key] = value
			}
		}

		// Set the storage type explicitly
		storageConfig["type"] = storageType

		// Override with values from the StorageRef if provided
		if storage.Keep > 0 {
			storageConfig["keep"] = storage.Keep
		}
		if storage.Timeout > 0 {
			storageConfig["timeout"] = storage.Timeout
		}

		// Add to storages map
		storages[storage.Name] = storageConfig
	}

	// Create the model
	backupModel := Model{
		Databases: databases,
		Storages:  storages,
	}

	// Add optional fields if provided
	if model.BeforeScript != "" {
		backupModel.BeforeScript = model.BeforeScript
	}
	if model.AfterScript != "" {
		backupModel.AfterScript = model.AfterScript
	}

	// Add compression if specified
	if model.CompressWith != nil && model.CompressWith.Type != "" {
		backupModel.Compress = model.CompressWith.Type
	}

	// Add encoding if specified
	if model.EncodeWith != nil && model.EncodeWith.Type != "" {
		backupModel.Encode = model.EncodeWith.Type
	}

	// Create the backup config
	backupConfig := BackupConfig{
		Models: map[string]Model{
			name: backupModel,
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&backupConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal backup config: %w", err)
	}

	// Create the Secret object
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"gobackup.yml": string(yamlData),
		},
	}

	// Check if the secret already exists
	found, err := k.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the Secret
			_, err = k.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing secret: %w", err)
	}

	// Update the existing secret
	secret.ResourceVersion = found.ResourceVersion
	_, err = k.Clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// DeleteSecret deletes a secret
func (k *K8s) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := k.Clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
