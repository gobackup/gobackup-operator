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

// Package gobackupconfig renders a Backup (plus its referenced Database and
// Storage resources) into the gobackup.yml document consumed by the gobackup
// tool.
//
// The rendering is deliberately lossless: each Database/Storage config is
// projected to a map through its JSON tags, so the Go struct tag *is* the
// gobackup key. Adding a field to DatabaseConfig/StorageConfig cannot be
// silently dropped, and there is no per-field switch to keep in sync. A
// golden-test suite and a contract test (against the gobackup source) guard
// this property.
package gobackupconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
)

// BackupConfig is the root of a gobackup.yml document.
type BackupConfig struct {
	Models map[string]Model `json:"models"`
}

// Model is a single gobackup model (one per Backup resource).
type Model struct {
	Databases    map[string]map[string]any `json:"databases,omitempty"`
	Storages     map[string]map[string]any `json:"storages,omitempty"`
	BeforeScript string                    `json:"before_script,omitempty"`
	AfterScript  string                    `json:"after_script,omitempty"`
	Compress     string                    `json:"compress_with,omitempty"`
	Encode       string                    `json:"encode_with,omitempty"`
}

// SecretResolver resolves a Secret name/key pair in a namespace to its
// plaintext value. Build calls it for every "<field>_ref" config field.
type SecretResolver func(ctx context.Context, namespace, name, key string) (string, error)

// Build renders the gobackup.yml bytes for backup, using the referenced
// Database/Storage resources (looked up by name) and resolving any Secret
// references via resolve. It is pure: given the same inputs and resolver it
// always produces the same bytes, which is what the golden tests assert.
func Build(
	ctx context.Context,
	backup *backupv1.Backup,
	dbByName map[string]*backupv1.Database,
	storageByName map[string]*backupv1.Storage,
	resolve SecretResolver,
) ([]byte, error) {
	if backup == nil {
		return nil, fmt.Errorf("backup cannot be nil")
	}

	model := Model{
		Databases: map[string]map[string]any{},
		Storages:  map[string]map[string]any{},
	}
	namespace := backup.Namespace

	for _, ref := range backup.Spec.DatabaseRefs {
		db, ok := dbByName[ref.Name]
		if !ok || db == nil {
			return nil, fmt.Errorf("database %q referenced by backup %q was not provided", ref.Name, backup.Name)
		}
		dbType := strings.ToLower(strings.TrimSpace(db.Spec.Type))
		if dbType == "" {
			return nil, fmt.Errorf("database %q has an empty spec.type", ref.Name)
		}
		entry, err := subConfigToMap(db.Spec.Config, dbType)
		if err != nil {
			return nil, fmt.Errorf("marshal database %q config: %w", ref.Name, err)
		}
		if err := resolveRefs(ctx, namespace, entry, resolve); err != nil {
			return nil, fmt.Errorf("resolve secrets for database %q: %w", ref.Name, err)
		}
		entry["type"] = dbType
		model.Databases[ref.Name] = entry
	}

	for _, ref := range backup.Spec.StorageRefs {
		storage, ok := storageByName[ref.Name]
		if !ok || storage == nil {
			return nil, fmt.Errorf("storage %q referenced by backup %q was not provided", ref.Name, backup.Name)
		}
		// One source of truth for the backend type: the Storage resource,
		// mirroring the Database path (fixes the historical asymmetry where
		// storage type came from the ref).
		storageType := strings.ToLower(strings.TrimSpace(storage.Spec.Type))
		if storageType == "" {
			return nil, fmt.Errorf("storage %q has an empty spec.type", ref.Name)
		}
		entry, err := subConfigToMap(storage.Spec.Config, storageType)
		if err != nil {
			return nil, fmt.Errorf("marshal storage %q config: %w", ref.Name, err)
		}
		if err := resolveRefs(ctx, namespace, entry, resolve); err != nil {
			return nil, fmt.Errorf("resolve secrets for storage %q: %w", ref.Name, err)
		}
		entry["type"] = storageType
		// Per-backup retention/timeout overrides carried on the ref.
		if ref.Keep > 0 {
			entry["keep"] = ref.Keep
		}
		if ref.Timeout > 0 {
			entry["timeout"] = ref.Timeout
		}
		model.Storages[ref.Name] = entry
	}

	if backup.Spec.BeforeScript != "" {
		model.BeforeScript = backup.Spec.BeforeScript
	}
	if backup.Spec.AfterScript != "" {
		model.AfterScript = backup.Spec.AfterScript
	}
	if backup.Spec.CompressWith != nil && backup.Spec.CompressWith.Type != "" {
		model.Compress = backup.Spec.CompressWith.Type
	}
	if backup.Spec.EncodeWith != nil && backup.Spec.EncodeWith.Type != "" {
		model.Encode = backup.Spec.EncodeWith.Type
	}

	cfg := BackupConfig{Models: map[string]Model{backup.Name: model}}
	return yaml.Marshal(cfg)
}

// subConfigToMap projects the nested config sub-object matching backendType to a
// flat map keyed by its JSON tags. The CRD nests one sub-object per backend type
// under config (config.postgresql, config.s3, …) purely as authoring sugar; the
// emitted gobackup.yml stays flat, so we serialize the whole config, select the
// sub-object keyed by backendType, and return its (already flat) contents. The
// JSON tag on each sub-struct field is the gobackup key, so nothing is dropped.
func subConfigToMap(config any, backendType string) (map[string]any, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	full := map[string]any{}
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, err
	}
	sub, ok := full[backendType]
	if !ok {
		return nil, fmt.Errorf("config.%s is not set", backendType)
	}
	entry, ok := sub.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config.%s is not an object", backendType)
	}
	return entry, nil
}

// resolveRefs replaces every "<field>_ref" entry (a serialized
// SecretKeySelector) with its plaintext value under "<field>".
func resolveRefs(ctx context.Context, namespace string, entry map[string]any, resolve SecretResolver) error {
	for key, value := range entry {
		if !strings.HasSuffix(key, "_ref") {
			continue
		}
		sel, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("secret reference %q is not an object", key)
		}
		name, _ := sel["name"].(string)
		secretKey, _ := sel["key"].(string)
		if name == "" || secretKey == "" {
			return fmt.Errorf("secret reference %q missing name or key", key)
		}
		if resolve == nil {
			return fmt.Errorf("secret reference %q present but no resolver configured", key)
		}
		val, err := resolve(ctx, namespace, name, secretKey)
		if err != nil {
			return err
		}
		entry[strings.TrimSuffix(key, "_ref")] = val
		delete(entry, key)
	}
	return nil
}
