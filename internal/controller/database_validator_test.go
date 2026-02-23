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
	"testing"
)

func TestValidateDatabaseConfig_Redis(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid mode copy",
			config:  map[string]interface{}{"mode": "copy"},
			wantErr: false,
		},
		{
			name:    "valid mode sync",
			config:  map[string]interface{}{"mode": "sync"},
			wantErr: false,
		},
		{
			name:    "invalid mode",
			config:  map[string]interface{}{"mode": "dump"},
			wantErr: true,
			errMsg:  "redis 'mode' must be 'copy' or 'sync', got 'dump'",
		},
		{
			name:    "mode is not a string",
			config:  map[string]interface{}{"mode": 42},
			wantErr: true,
			errMsg:  "redis 'mode' must be a string",
		},
		{
			name:    "no mode set is valid",
			config:  map[string]interface{}{"host": "localhost"},
			wantErr: false,
		},
		{
			name:    "redis-only fields are allowed for redis",
			config:  map[string]interface{}{"mode": "copy", "rdb_path": "/var/lib/redis/dump.rdb", "invoke_save": true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseConfig("redis", tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateDatabaseConfig_NonRedis(t *testing.T) {
	tests := []struct {
		name    string
		dbType  string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "postgresql with mode set",
			dbType:  "postgresql",
			config:  map[string]interface{}{"mode": "copy"},
			wantErr: true,
			errMsg:  "field 'mode' is only valid for Redis databases, not for 'postgresql'",
		},
		{
			name:    "mysql with mode set",
			dbType:  "mysql",
			config:  map[string]interface{}{"mode": "sync"},
			wantErr: true,
			errMsg:  "field 'mode' is only valid for Redis databases, not for 'mysql'",
		},
		{
			name:    "postgresql with rdb_path set",
			dbType:  "postgresql",
			config:  map[string]interface{}{"rdb_path": "/var/lib/redis/dump.rdb"},
			wantErr: true,
			errMsg:  "field 'rdb_path' is only valid for Redis databases, not for 'postgresql'",
		},
		{
			name:    "mongodb with invoke_save set",
			dbType:  "mongodb",
			config:  map[string]interface{}{"invoke_save": true},
			wantErr: true,
			errMsg:  "field 'invoke_save' is only valid for Redis databases, not for 'mongodb'",
		},
		{
			name:    "postgresql with valid fields",
			dbType:  "postgresql",
			config:  map[string]interface{}{"host": "localhost", "database": "mydb", "username": "admin"},
			wantErr: false,
		},
		{
			name:    "mysql with valid fields",
			dbType:  "mysql",
			config:  map[string]interface{}{"host": "localhost", "username": "root"},
			wantErr: false,
		},
		{
			name:    "mariadb with valid fields",
			dbType:  "mariadb",
			config:  map[string]interface{}{"host": "localhost"},
			wantErr: false,
		},
		{
			name:    "mongodb with valid fields",
			dbType:  "mongodb",
			config:  map[string]interface{}{"host": "localhost", "auth_db": "admin"},
			wantErr: false,
		},
		{
			name:    "mssql with valid fields",
			dbType:  "mssql",
			config:  map[string]interface{}{"host": "localhost", "trust_server_certificate": true},
			wantErr: false,
		},
		{
			name:    "influxdb with valid fields",
			dbType:  "influxdb",
			config:  map[string]interface{}{"host": "localhost", "token": "mytoken", "bucket": "mybucket"},
			wantErr: false,
		},
		{
			name:    "etcd with valid fields",
			dbType:  "etcd",
			config:  map[string]interface{}{"endpoints": []string{"localhost:2379"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseConfig(tt.dbType, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateDatabaseConfig_EmptyConfig(t *testing.T) {
	// Empty config should be valid for any database type
	for _, dbType := range []string{"postgresql", "mysql", "mariadb", "mongodb", "redis", "mssql", "influxdb", "etcd"} {
		t.Run(dbType, func(t *testing.T) {
			err := validateDatabaseConfig(dbType, map[string]interface{}{})
			if err != nil {
				t.Errorf("empty config for %s should be valid, got error: %v", dbType, err)
			}
		})
	}
}
