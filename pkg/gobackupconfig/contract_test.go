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
	"reflect"
	"strings"
	"testing"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
)

// The config keys each gobackup v3.1.0 backend actually reads, extracted
// per-backend from the gobackup source:
//
//	grep -oE 'viper\.Get[A-Za-z]*\("[^"]+"\)' database/<backend>.go storage/<backend>.go
//
// plus `keep` (read by every storage backend via storage/base.go's cycler).
//
// This is the authoritative contract. After E3 each backend has its own nested
// config sub-struct, so every non-secret-ref json tag on that sub-struct must be
// a key THAT backend reads — otherwise the operator would emit a key gobackup
// silently ignores (a field-drop bug). Full parity means the allowlists below
// are EMPTY: any mismatch is a hard failure.
var gobackupDatabaseKeys = map[string]map[string]bool{
	"postgresql": keySet("host", "port", "socket", "database", "username", "password", "tables", "exclude_tables", "args", "compress", "all_databases"),
	"mysql":      keySet("host", "port", "socket", "database", "username", "password", "tables", "exclude_tables", "args", "all_databases"),
	"mariadb":    keySet("host", "port", "socket", "database", "username", "password", "args", "all_databases"),
	"mongodb":    keySet("host", "port", "database", "username", "password", "authdb", "oplog", "exclude_tables", "exclude_tables_prefix", "uri", "args", "all_databases"),
	"redis":      keySet("host", "port", "socket", "password", "mode", "invoke_save", "rdb_path", "args"),
	"mssql":      keySet("host", "port", "database", "username", "password", "trust_server_certificate", "skip_databases", "args", "all_databases"),
	"influxdb":   keySet("host", "token", "org", "org_id", "bucket", "bucket_id", "skip_verify", "http_debug", "all_databases"),
	"etcd":       keySet("endpoint", "endpoints", "args"),
	"firebird":   keySet("host", "port", "database", "username", "password", "role", "args"),
	"sqlite":     keySet("path"),
}

// s3Keys is shared by every S3-family backend (s3, oss, r2, spaces, b2, cos,
// us3, kodo, bos, minio, obs, tos, upyun); they use the same gobackup s3 driver.
var s3Keys = keySet("bucket", "region", "endpoint", "path", "access_key_id", "secret_access_key", "token", "storage_class", "max_retries", "force_path_style", "account_id", "timeout", "keep")

var gobackupStorageKeys = map[string]map[string]bool{
	"local":  keySet("path"),
	"s3":     s3Keys,
	"oss":    s3Keys,
	"r2":     s3Keys,
	"spaces": s3Keys,
	"b2":     s3Keys,
	"cos":    s3Keys,
	"us3":    s3Keys,
	"kodo":   s3Keys,
	"bos":    s3Keys,
	"minio":  s3Keys,
	"obs":    s3Keys,
	"tos":    s3Keys,
	"upyun":  s3Keys,
	"gcs":    keySet("bucket", "path", "credentials", "credentials_file", "timeout", "keep"),
	"azure":  keySet("account", "container", "bucket", "path", "client_id", "client_secret", "tenant_id", "timeout", "keep"),
	"ftp":    keySet("host", "port", "path", "username", "password", "tls", "explicit_tls", "no_check_certificate", "timeout", "keep"),
	"sftp":   keySet("host", "port", "path", "username", "password", "private_key", "passphrase", "timeout", "keep"),
	"scp":    keySet("host", "port", "path", "username", "password", "private_key", "passphrase", "timeout", "keep"),
	"webdav": keySet("path", "root", "username", "password", "keep"),
}

func TestDatabaseConfigMatchesGobackupContract(t *testing.T) {
	assertContract(t, reflect.TypeOf(backupv1.DatabaseConfig{}), gobackupDatabaseKeys, "DatabaseConfig")
}

func TestStorageConfigMatchesGobackupContract(t *testing.T) {
	assertContract(t, reflect.TypeOf(backupv1.StorageConfig{}), gobackupStorageKeys, "StorageConfig")
}

// assertContract reflects over the per-backend config struct (DatabaseConfig or
// StorageConfig), whose every field is a pointer to a backend sub-struct tagged
// with the backend name (e.g. `postgresql`, `s3`). For each backend it verifies
// that (a) the backend is present in the authoritative key map and (b) every
// non-"_ref" json tag on the sub-struct is a real gobackup key for that backend.
// There is no allowlist: a mismatch fails the test — the definition of done for
// full gobackup v3.1.0 parity.
func assertContract(t *testing.T, cfgType reflect.Type, keysByBackend map[string]map[string]bool, name string) {
	t.Helper()

	covered := map[string]bool{}
	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		backend := jsonTag(field)
		if backend == "" || backend == "-" {
			continue
		}
		valid, ok := keysByBackend[backend]
		if !ok {
			t.Errorf("%s: backend %q has no authoritative gobackup key set", name, backend)
			continue
		}
		covered[backend] = true

		sub := field.Type
		for sub.Kind() == reflect.Pointer {
			sub = sub.Elem()
		}
		if sub.Kind() != reflect.Struct {
			t.Errorf("%s: backend %q field is not a struct pointer", name, backend)
			continue
		}
		for j := 0; j < sub.NumField(); j++ {
			tag := jsonTag(sub.Field(j))
			if tag == "" || tag == "-" || strings.HasSuffix(tag, "_ref") {
				continue // untagged, ignored, or operator-only secret sugar
			}
			if !valid[tag] {
				t.Errorf("%s.%s: json tag %q is NOT a gobackup key for the %q backend — it would be silently dropped by gobackup.", name, backend, tag, backend)
			}
		}
	}

	// Guard against a key map that references a backend the struct dropped.
	for backend := range keysByBackend {
		if !covered[backend] {
			t.Errorf("%s: key map lists backend %q but no config field emits it", name, backend)
		}
	}
}

func jsonTag(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func keySet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}
