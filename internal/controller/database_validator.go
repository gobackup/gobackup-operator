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

package controller

import (
	"context"
	"fmt"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
)

// redisOnlyFields lists fields that are only valid for Redis databases.
var redisOnlyFields = []string{"mode", "rdb_path", "invoke_save"}

// validRedisModes contains the allowed values for the Redis mode field.
var validRedisModes = map[string]bool{
	"copy": true,
	"sync": true,
}

// validateDatabaseConfig validates the configuration for a database based on its type.
// It checks that:
//   - For Redis: the mode field, if set, is either "copy" or "sync"
//   - For non-Redis databases: Redis-specific fields (mode, rdb_path, invoke_save) are not set
func validateDatabaseConfig(dbType string, config map[string]interface{}) error {
	if dbType == "redis" {
		if mode, ok := config["mode"]; ok {
			modeStr, ok := mode.(string)
			if !ok {
				return fmt.Errorf("redis 'mode' must be a string")
			}
			if !validRedisModes[modeStr] {
				return fmt.Errorf("redis 'mode' must be 'copy' or 'sync', got '%s'", modeStr)
			}
		}
	} else {
		for _, field := range redisOnlyFields {
			if _, ok := config[field]; ok {
				return fmt.Errorf("field '%s' is only valid for Redis databases, not for '%s'", field, dbType)
			}
		}
	}
	return nil
}

// validateDatabaseRefs fetches and validates the configuration of all Database CRDs
// referenced in the backup spec.
func (r *BackupReconciler) validateDatabaseRefs(ctx context.Context, backup *backupv1.Backup) error {
	for _, dbRef := range backup.Spec.DatabaseRefs {
		apiGroup := dbRef.APIGroup
		if apiGroup == "" {
			apiGroup = "gobackup.io"
		}

		databaseCRD, err := r.K8s.GetCRD(ctx, apiGroup, "v1", "databases", backup.Namespace, dbRef.Name)
		if err != nil {
			return fmt.Errorf("failed to get database %s: %w", dbRef.Name, err)
		}

		specMap, ok := databaseCRD.Object["spec"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("database spec for %s is not a valid map", dbRef.Name)
		}

		dbType, ok := specMap["type"].(string)
		if !ok || dbType == "" {
			return fmt.Errorf("database type for %s is missing or invalid", dbRef.Name)
		}

		configMap, _ := specMap["config"].(map[string]interface{})
		if configMap == nil {
			configMap = make(map[string]interface{})
		}

		if err := validateDatabaseConfig(dbType, configMap); err != nil {
			return fmt.Errorf("invalid configuration for database %s: %w", dbRef.Name, err)
		}
	}
	return nil
}
