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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of Database
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.mode)",message="config.mode is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.sync)",message="config.sync is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.copy)",message="config.copy is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.invoke_save)",message="config.invoke_save is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.rdb_path)",message="config.rdb_path is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.args_redis)",message="config.args_redis is only valid when spec.type is redis"
// +kubebuilder:validation:XValidation:rule="self.type == 'mongodb' || !has(self.config.auth_db)",message="config.auth_db is only valid when spec.type is mongodb"
// +kubebuilder:validation:XValidation:rule="self.type == 'mongodb' || !has(self.config.oplog)",message="config.oplog is only valid when spec.type is mongodb"
// +kubebuilder:validation:XValidation:rule="self.type == 'mssql' || !has(self.config.trust_server_certificate)",message="config.trust_server_certificate is only valid when spec.type is mssql"
// +kubebuilder:validation:XValidation:rule="self.type == 'influxdb' || !has(self.config.token)",message="config.token is only valid when spec.type is influxdb"
// +kubebuilder:validation:XValidation:rule="self.type == 'influxdb' || !has(self.config.bucket)",message="config.bucket is only valid when spec.type is influxdb"
// +kubebuilder:validation:XValidation:rule="self.type == 'influxdb' || !has(self.config.org)",message="config.org is only valid when spec.type is influxdb"
// +kubebuilder:validation:XValidation:rule="self.type == 'etcd' || !has(self.config.endpoints)",message="config.endpoints is only valid when spec.type is etcd"
// +kubebuilder:validation:XValidation:rule="self.type in ['postgresql', 'mysql', 'mariadb', 'mssql'] || !has(self.config.tables)",message="config.tables is only valid for SQL databases (postgresql, mysql, mariadb, mssql)"
// +kubebuilder:validation:XValidation:rule="self.type in ['postgresql', 'mysql', 'mariadb', 'mssql'] || !has(self.config.exclude_tables)",message="config.exclude_tables is only valid for SQL databases (postgresql, mysql, mariadb, mssql)"
type DatabaseSpec struct {
	// Type is the database backend type
	// +kubebuilder:validation:Enum=postgresql;mysql;mariadb;mongodb;redis;mssql;influxdb;etcd
	Type string `json:"type"`

	// Config contains the database configuration
	Config DatabaseConfig `json:"config"`
}

// DatabaseConfig defines the configuration for all database types
type DatabaseConfig struct {
	// Common fields applicable to all or multiple database types

	// Host is the database server hostname
	// Default for PostgreSQL: localhost, for Redis: 127.0.0.1
	Host *string `json:"host,omitempty"`

	// Port is the database server port
	// Default for PostgreSQL: 5432, for Redis: 6379
	Port *int `json:"port,omitempty"`

	// Socket is the database server socket
	// For PostgreSQL: e.g. /var/run/postgresql/.s.PGSQL.5432
	// For Redis: e.g. /var/run/redis/redis.sock
	Socket *string `json:"socket,omitempty"`

	// Password is the password for the database or Redis server
	// Default for Redis: ""
	Password *string `json:"password,omitempty"`

	// Args are additional arguments for pg_dump (PostgreSQL), mysqldump (MySQL) or redis-cli utility (Redis)
	// For Redis, e.g.: --tls --cacert redis_ca.pem
	// For MySQL, e.g.: --skip-ssl or --ssl-ca=/path/to/ca.pem
	Args *string `json:"args,omitempty"`

	// PostgreSQL-specific fields

	// Database is the database name (PostgreSQL)
	Database *string `json:"database,omitempty"`

	// Username is the username for the database (PostgreSQL)
	// Default: root
	Username *string `json:"username,omitempty"`

	// Tables is an array of tables to backup (PostgreSQL)
	Tables []string `json:"tables,omitempty"`

	// ExcludeTables is an array of tables to exclude from backup (PostgreSQL)
	ExcludeTables []string `json:"exclude_tables,omitempty"`

	// MongoDB-specific fields

	// AuthDB is the authentication database (MongoDB)
	AuthDB *string `json:"auth_db,omitempty"`

	// Oplog is used to backup oplog (MongoDB)
	Oplog *bool `json:"oplog,omitempty"`

	// MSSQL-specific fields

	// TrustServerCertificate is used to trust the server certificate (MSSQL)
	TrustServerCertificate *bool `json:"trust_server_certificate,omitempty"`

	// InfluxDB-specific fields

	// Token is the authentication token (InfluxDB)
	Token *string `json:"token,omitempty"`

	// Bucket is the bucket name (InfluxDB)
	Bucket *string `json:"bucket,omitempty"`

	// Organization is the organization name (InfluxDB)
	Organization *string `json:"org,omitempty"`

	// ETCD-specific fields

	// Endpoints are the ETCD endpoints (ETCD)
	Endpoints []string `json:"endpoints,omitempty"`

	// Redis-specific fields

	// Mode is the Redis dump mode. Default: copy
	// +kubebuilder:validation:Enum=copy;sync
	Mode *string `json:"mode,omitempty"`

	// Sync is used for remote Redis server to export
	Sync *bool `json:"sync,omitempty"`

	// Copy is used for local Redis server, just copy Redis dump.db
	Copy *bool `json:"copy,omitempty"`

	// InvokeSave invokes save before backup (Redis). Default: true
	InvokeSave *bool `json:"invoke_save,omitempty"`

	// RdbPath is the path to dump.rdb for Redis. Default: /var/lib/redis/dump.rdb
	RdbPath *string `json:"rdb_path,omitempty"`

	// ArgsRedis are additional options for redis-cli utility, for example: --tls --cacert redis_ca.pem
	ArgsRedis *string `json:"args_redis,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:resource:shortName=db
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
