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

package gobackupconfig

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
)

var update = flag.Bool("update", false, "update golden files")

// ptr returns a pointer to v (test helper for the pointer-heavy config structs).
func ptr[T any](v T) *T { return &v }

// secretRef builds a SecretKeySelector referencing name/key.
func secretRef(name, key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: name},
		Key:                  key,
	}
}

// fakeResolver returns a deterministic plaintext for any Secret name/key so
// golden output is stable and no Kubernetes client is needed.
func fakeResolver(_ context.Context, _, name, key string) (string, error) {
	return "SECRET(" + name + "/" + key + ")", nil
}

func meta(name string) metav1.ObjectMeta { return metav1.ObjectMeta{Name: name, Namespace: "default"} }

func TestBuildGolden(t *testing.T) {
	cases := []struct {
		name     string
		backup   *backupv1.Backup
		dbs      map[string]*backupv1.Database
		storages map[string]*backupv1.Storage
	}{
		{
			name: "postgresql_plaintext_local",
			backup: &backupv1.Backup{
				ObjectMeta: meta("app-backup"),
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{{Name: "app-pg"}},
					StorageRefs:  []backupv1.StorageRef{{Name: "app-local"}},
					CompressWith: &backupv1.Compress{Type: "gzip"},
				},
			},
			dbs: map[string]*backupv1.Database{
				"app-pg": {ObjectMeta: meta("app-pg"), Spec: backupv1.DatabaseSpec{
					Type: "postgresql",
					Config: backupv1.DatabaseConfig{
						PostgreSQL: &backupv1.PostgreSQLConfig{
							Host:          ptr("localhost"),
							Port:          ptr(5432),
							Database:      ptr("appdb"),
							Username:      ptr("appuser"),
							Password:      ptr("s3cr3t"),
							Tables:        []string{"users", "orders"},
							ExcludeTables: []string{"audit"},
							Args:          ptr("--no-owner"),
						},
					},
				}},
			},
			storages: map[string]*backupv1.Storage{
				"app-local": {ObjectMeta: meta("app-local"), Spec: backupv1.StorageSpec{
					Type:   "local",
					Config: backupv1.StorageConfig{Local: &backupv1.LocalConfig{Path: ptr("/backups")}},
				}},
			},
		},
		{
			name: "postgresql_secretref_s3",
			backup: &backupv1.Backup{
				ObjectMeta: meta("app-backup"),
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{{Name: "app-pg"}},
					StorageRefs:  []backupv1.StorageRef{{Name: "app-s3", Keep: 30, Timeout: 600}},
				},
			},
			dbs: map[string]*backupv1.Database{
				"app-pg": {ObjectMeta: meta("app-pg"), Spec: backupv1.DatabaseSpec{
					Type: "postgresql",
					Config: backupv1.DatabaseConfig{
						PostgreSQL: &backupv1.PostgreSQLConfig{
							Host:        ptr("pg.svc"),
							Port:        ptr(5432),
							Database:    ptr("appdb"),
							UsernameRef: secretRef("db-creds", "username"),
							PasswordRef: secretRef("db-creds", "password"),
						},
					},
				}},
			},
			storages: map[string]*backupv1.Storage{
				"app-s3": {ObjectMeta: meta("app-s3"), Spec: backupv1.StorageSpec{
					Type: "s3",
					Config: backupv1.StorageConfig{
						S3: &backupv1.S3CompatibleConfig{
							Bucket:             ptr("my-bucket"),
							Region:             ptr("us-east-1"),
							Path:               ptr("backups/app"),
							AccessKeyIDRef:     secretRef("s3-creds", "access-key-id"),
							SecretAccessKeyRef: secretRef("s3-creds", "secret-access-key"),
							StorageClass:       ptr("STANDARD_IA"),
						},
					},
				}},
			},
		},
		{
			name: "redis_minio",
			backup: &backupv1.Backup{
				ObjectMeta: meta("cache-backup"),
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{{Name: "cache"}},
					StorageRefs:  []backupv1.StorageRef{{Name: "minio"}},
				},
			},
			dbs: map[string]*backupv1.Database{
				"cache": {ObjectMeta: meta("cache"), Spec: backupv1.DatabaseSpec{
					Type: "redis",
					Config: backupv1.DatabaseConfig{
						// Proves the redis bug-fix: only mode + args are emitted
						// (never sync/copy/args_redis).
						Redis: &backupv1.RedisConfig{
							Host:       ptr("redis.svc"),
							Port:       ptr(6379),
							Password:   ptr("redispw"),
							Mode:       ptr("copy"),
							InvokeSave: ptr(true),
							Args:       ptr("--tls"),
						},
					},
				}},
			},
			storages: map[string]*backupv1.Storage{
				"minio": {ObjectMeta: meta("minio"), Spec: backupv1.StorageSpec{
					Type: "minio",
					Config: backupv1.StorageConfig{
						MinIO: &backupv1.S3CompatibleConfig{
							Bucket:          ptr("dumps"),
							Endpoint:        ptr("http://minio.svc:9000"),
							AccessKeyID:     ptr("minioadmin"),
							SecretAccessKey: ptr("minioadmin"),
						},
					},
				}},
			},
		},
		{
			name: "multi_full",
			backup: &backupv1.Backup{
				ObjectMeta: meta("full"),
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{{Name: "pg"}, {Name: "mongo"}},
					StorageRefs:  []backupv1.StorageRef{{Name: "s3", Keep: 7}, {Name: "local"}},
					BeforeScript: "echo before",
					AfterScript:  "echo after",
					CompressWith: &backupv1.Compress{Type: "gzip"},
					EncodeWith:   &backupv1.Encode{Type: "base64"},
				},
			},
			dbs: map[string]*backupv1.Database{
				"pg": {ObjectMeta: meta("pg"), Spec: backupv1.DatabaseSpec{
					Type:   "postgresql",
					Config: backupv1.DatabaseConfig{PostgreSQL: &backupv1.PostgreSQLConfig{Host: ptr("pg"), Port: ptr(5432), Database: ptr("d")}},
				}},
				// Proves the mongodb bug-fix: auth DB is emitted as `authdb`.
				"mongo": {ObjectMeta: meta("mongo"), Spec: backupv1.DatabaseSpec{
					Type:   "mongodb",
					Config: backupv1.DatabaseConfig{MongoDB: &backupv1.MongoDBConfig{Host: ptr("mongo"), Port: ptr(27017), Database: ptr("m"), AuthDB: ptr("admin"), Oplog: ptr(true)}},
				}},
			},
			storages: map[string]*backupv1.Storage{
				"s3":    {ObjectMeta: meta("s3"), Spec: backupv1.StorageSpec{Type: "s3", Config: backupv1.StorageConfig{S3: &backupv1.S3CompatibleConfig{Bucket: ptr("b"), Region: ptr("r")}}}},
				"local": {ObjectMeta: meta("local"), Spec: backupv1.StorageSpec{Type: "local", Config: backupv1.StorageConfig{Local: &backupv1.LocalConfig{Path: ptr("/b")}}}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(context.Background(), tc.backup, tc.dbs, tc.storages, fakeResolver)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			golden := filepath.Join("testdata", tc.name+".golden.yml")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("gobackup.yml mismatch for %s.\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}

// TestBuildRegressionGuard proves the headline fix: a config field that is set
// appears in the output with NO change to secret.go / the emitter. If this
// fails, a field is being silently dropped.
func TestBuildRegressionGuard(t *testing.T) {
	backup := &backupv1.Backup{
		ObjectMeta: meta("b"),
		Spec: backupv1.BackupSpec{
			DatabaseRefs: []backupv1.DatabaseRef{{Name: "d"}},
		},
	}
	dbs := map[string]*backupv1.Database{
		"d": {ObjectMeta: meta("d"), Spec: backupv1.DatabaseSpec{
			Type:   "postgresql",
			Config: backupv1.DatabaseConfig{PostgreSQL: &backupv1.PostgreSQLConfig{ExcludeTables: []string{"secret_table"}}},
		}},
	}
	got, err := Build(context.Background(), backup, dbs, nil, fakeResolver)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(string(got), "exclude_tables") || !strings.Contains(string(got), "secret_table") {
		t.Errorf("exclude_tables was silently dropped from output:\n%s", got)
	}
}
