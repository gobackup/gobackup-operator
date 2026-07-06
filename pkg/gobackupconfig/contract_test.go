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

// The config keys each gobackup v3.1.0 backend actually reads, extracted from
// the gobackup source:
//
//	grep -oE 'viper\.Get[A-Za-z]*\("[^"]+"\)' database/*.go storage/*.go
//
// This is the authoritative contract. Every non-secret-ref field of
// DatabaseConfig/StorageConfig must serialize to a key some backend reads,
// otherwise the operator emits a key gobackup silently ignores (a field-drop
// bug). Keys read by all/most backends (e.g. storage `keep`, read by the
// cycler) are included.
var gobackupDatabaseKeys = keySet(
	// common / SQL
	"host", "port", "socket", "database", "username", "password",
	"tables", "exclude_tables", "args", "compress", "all_databases",
	// mongodb
	"authdb", "oplog", "exclude_tables_prefix", "uri",
	// redis
	"mode", "invoke_save", "rdb_path",
	// mssql
	"trust_server_certificate", "skip_databases",
	// influxdb2
	"token", "org", "org_id", "bucket", "bucket_id", "skip_verify", "http_debug",
	// etcd
	"endpoint", "endpoints",
	// firebird
	"role",
)

var gobackupStorageKeys = keySet(
	// common (path/timeout per backend; keep via cycler)
	"path", "timeout", "keep",
	// auth
	"username", "password", "private_key", "passphrase",
	// s3 family
	"bucket", "region", "endpoint", "access_key_id", "secret_access_key",
	"token", "storage_class", "max_retries", "force_path_style", "account_id",
	// gcs
	"credentials", "credentials_file",
	// azure
	"account", "container", "tenant_id", "client_id", "client_secret",
	// ftp/sftp/scp/webdav
	"host", "port", "root", "tls", "explicit_tls", "no_check_certificate",
)

// knownMismatches are CRD json tags that do NOT match a real gobackup key today.
// These are the confirmed silent-drop bugs; they are fixed (and this allowlist
// emptied) in E3, where the per-backend restructure aligns every field with the
// gobackup source of truth. See issue #80.
var knownDatabaseMismatches = keySet(
	"auth_db",    // gobackup reads "authdb"
	"args_redis", // gobackup reads "args" for redis
	"sync",       // gobackup only honors "mode: copy|sync"
	"copy",       // gobackup only honors "mode: copy|sync"
)

var knownStorageMismatches = keySet( /* empty: storage is already aligned */ )

func TestDatabaseConfigMatchesGobackupContract(t *testing.T) {
	assertContract(t, reflect.TypeOf(backupv1.DatabaseConfig{}), gobackupDatabaseKeys, knownDatabaseMismatches, "DatabaseConfig")
}

func TestStorageConfigMatchesGobackupContract(t *testing.T) {
	assertContract(t, reflect.TypeOf(backupv1.StorageConfig{}), gobackupStorageKeys, knownStorageMismatches, "StorageConfig")
}

// assertContract fails if any non-"_ref" json tag on typ is neither a real
// gobackup key nor an explicitly-allowlisted known mismatch. It also fails if
// the allowlist has grown stale (an entry that now matches or no longer exists).
func assertContract(t *testing.T, typ reflect.Type, valid, allow map[string]bool, name string) {
	t.Helper()
	seen := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		tag := jsonTag(typ.Field(i))
		if tag == "" || tag == "-" || strings.HasSuffix(tag, "_ref") {
			continue // untagged, ignored, or operator-only secret sugar
		}
		seen[tag] = true
		if valid[tag] {
			if allow[tag] {
				t.Errorf("%s: json tag %q is a valid gobackup key but is still in the allowlist; remove it", name, tag)
			}
			continue
		}
		if allow[tag] {
			continue // known bug, tracked for E3
		}
		t.Errorf("%s: json tag %q is NOT a gobackup key and not allowlisted — it would be silently dropped by gobackup. Fix the tag or extend the contract.", name, tag)
	}
	for tag := range allow {
		if !seen[tag] {
			t.Errorf("%s: allowlist entry %q no longer exists as a field; remove it", name, tag)
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
