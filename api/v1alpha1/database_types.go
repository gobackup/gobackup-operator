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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of Database.
//
// Config carries one nested sub-object per database type; exactly the
// sub-object matching spec.type must be set. The two CEL rules below enforce
// that invariant (the emitter flattens the chosen sub-object into gobackup.yml).
//
// +kubebuilder:validation:XValidation:rule="(has(self.config.postgresql)?1:0)+(has(self.config.mysql)?1:0)+(has(self.config.mariadb)?1:0)+(has(self.config.mongodb)?1:0)+(has(self.config.redis)?1:0)+(has(self.config.mssql)?1:0)+(has(self.config.influxdb)?1:0)+(has(self.config.etcd)?1:0)+(has(self.config.firebird)?1:0)+(has(self.config.sqlite)?1:0) == 1",message="exactly one config sub-object must be set"
// +kubebuilder:validation:XValidation:rule="(self.type == 'postgresql' && has(self.config.postgresql)) || (self.type == 'mysql' && has(self.config.mysql)) || (self.type == 'mariadb' && has(self.config.mariadb)) || (self.type == 'mongodb' && has(self.config.mongodb)) || (self.type == 'redis' && has(self.config.redis)) || (self.type == 'mssql' && has(self.config.mssql)) || (self.type == 'influxdb' && has(self.config.influxdb)) || (self.type == 'etcd' && has(self.config.etcd)) || (self.type == 'firebird' && has(self.config.firebird)) || (self.type == 'sqlite' && has(self.config.sqlite))",message="the config sub-object must match spec.type"
type DatabaseSpec struct {
	// Type is the database backend type.
	// +kubebuilder:validation:Enum=postgresql;mysql;mariadb;mongodb;redis;mssql;influxdb;etcd;firebird;sqlite
	Type string `json:"type"`

	// Config carries exactly one nested sub-object matching spec.type.
	Config DatabaseConfig `json:"config"`
}

// DatabaseConfig holds one nested configuration sub-object per database type.
// Exactly the sub-object matching DatabaseSpec.Type is set; the others are nil.
type DatabaseConfig struct {
	// PostgreSQL configuration (spec.type: postgresql).
	// +optional
	PostgreSQL *PostgreSQLConfig `json:"postgresql,omitempty"`
	// MySQL configuration (spec.type: mysql).
	// +optional
	MySQL *MySQLConfig `json:"mysql,omitempty"`
	// MariaDB configuration (spec.type: mariadb).
	// +optional
	MariaDB *MariaDBConfig `json:"mariadb,omitempty"`
	// MongoDB configuration (spec.type: mongodb).
	// +optional
	MongoDB *MongoDBConfig `json:"mongodb,omitempty"`
	// Redis configuration (spec.type: redis).
	// +optional
	Redis *RedisConfig `json:"redis,omitempty"`
	// MSSQL configuration (spec.type: mssql).
	// +optional
	MSSQL *MSSQLConfig `json:"mssql,omitempty"`
	// InfluxDB configuration (spec.type: influxdb).
	// +optional
	InfluxDB *InfluxDBConfig `json:"influxdb,omitempty"`
	// Etcd configuration (spec.type: etcd).
	// +optional
	Etcd *EtcdConfig `json:"etcd,omitempty"`
	// Firebird configuration (spec.type: firebird).
	// +optional
	Firebird *FirebirdConfig `json:"firebird,omitempty"`
	// SQLite configuration (spec.type: sqlite).
	// +optional
	SQLite *SQLiteConfig `json:"sqlite,omitempty"`
}

// PostgreSQLConfig holds the gobackup keys for the postgresql backend.
type PostgreSQLConfig struct {
	// Host is the PostgreSQL server hostname.
	// +kubebuilder:default=localhost
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the PostgreSQL server port.
	// +kubebuilder:default=5432
	// +optional
	Port *int `json:"port,omitempty"`
	// Socket is the PostgreSQL server socket path.
	// +optional
	Socket *string `json:"socket,omitempty"`
	// Database is the database name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Tables restricts the dump to these tables.
	// +optional
	Tables []string `json:"tables,omitempty"`
	// ExcludeTables excludes these tables from the dump.
	// +optional
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	// Args are additional pg_dump arguments.
	// +optional
	Args *string `json:"args,omitempty"`
	// Compress passes the pg_dump compression option.
	// +optional
	Compress *string `json:"compress,omitempty"`
	// AllDatabases dumps all databases (pg_dumpall).
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// MySQLConfig holds the gobackup keys for the mysql backend.
type MySQLConfig struct {
	// Host is the MySQL server hostname.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the MySQL server port.
	// +kubebuilder:default=3306
	// +optional
	Port *int `json:"port,omitempty"`
	// Socket is the MySQL server socket path.
	// +optional
	Socket *string `json:"socket,omitempty"`
	// Database is the database name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +kubebuilder:default=root
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Tables restricts the dump to these tables.
	// +optional
	Tables []string `json:"tables,omitempty"`
	// ExcludeTables excludes these tables from the dump.
	// +optional
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	// Args are additional mysqldump arguments.
	// +optional
	Args *string `json:"args,omitempty"`
	// AllDatabases dumps all databases.
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// MariaDBConfig holds the gobackup keys for the mariadb backend.
type MariaDBConfig struct {
	// Host is the MariaDB server hostname.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the MariaDB server port.
	// +kubebuilder:default=3306
	// +optional
	Port *int `json:"port,omitempty"`
	// Socket is the MariaDB server socket path.
	// +optional
	Socket *string `json:"socket,omitempty"`
	// Database is the database name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +kubebuilder:default=root
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Args are additional mariadb-dump arguments.
	// +optional
	Args *string `json:"args,omitempty"`
	// AllDatabases dumps all databases.
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// MongoDBConfig holds the gobackup keys for the mongodb backend.
type MongoDBConfig struct {
	// Host is the MongoDB server hostname.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the MongoDB server port.
	// +kubebuilder:default=27017
	// +optional
	Port *int `json:"port,omitempty"`
	// Database is the database name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// AuthDB is the authentication database.
	// +optional
	AuthDB *string `json:"authdb,omitempty"`
	// Oplog enables a point-in-time oplog backup.
	// +optional
	Oplog *bool `json:"oplog,omitempty"`
	// ExcludeTables excludes these collections from the dump.
	// +optional
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	// ExcludeTablesPrefix excludes collections matching this prefix.
	// +optional
	ExcludeTablesPrefix *string `json:"exclude_tables_prefix,omitempty"`
	// URI is the full MongoDB connection URI.
	// +optional
	URI *string `json:"uri,omitempty"`
	// Args are additional mongodump arguments.
	// +optional
	Args *string `json:"args,omitempty"`
	// AllDatabases dumps all databases.
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// RedisConfig holds the gobackup keys for the redis backend.
type RedisConfig struct {
	// Host is the Redis server hostname.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the Redis server port.
	// +kubebuilder:default=6379
	// +optional
	Port *int `json:"port,omitempty"`
	// Socket is the Redis server socket path.
	// +optional
	Socket *string `json:"socket,omitempty"`
	// Password is the Redis password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the Redis password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Mode is the Redis dump mode.
	// +kubebuilder:validation:Enum=copy;sync
	// +kubebuilder:default=copy
	// +optional
	Mode *string `json:"mode,omitempty"`
	// InvokeSave runs SAVE before copying the dump file.
	// +kubebuilder:default=true
	// +optional
	InvokeSave *bool `json:"invoke_save,omitempty"`
	// RdbPath is the path to dump.rdb.
	// +optional
	RdbPath *string `json:"rdb_path,omitempty"`
	// Args are additional redis-cli arguments.
	// +optional
	Args *string `json:"args,omitempty"`
}

// MSSQLConfig holds the gobackup keys for the mssql backend.
type MSSQLConfig struct {
	// Host is the SQL Server hostname.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the SQL Server port.
	// +kubebuilder:default=1433
	// +optional
	Port *int `json:"port,omitempty"`
	// Database is the database name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +kubebuilder:default=sa
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// TrustServerCertificate trusts the server TLS certificate.
	// +optional
	TrustServerCertificate *bool `json:"trust_server_certificate,omitempty"`
	// SkipDatabases is a comma-separated list of databases to skip.
	// +optional
	SkipDatabases *string `json:"skip_databases,omitempty"`
	// Args are additional sqlcmd/bcp arguments.
	// +optional
	Args *string `json:"args,omitempty"`
	// AllDatabases dumps all databases.
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// InfluxDBConfig holds the gobackup keys for the influxdb (influxdb2) backend.
type InfluxDBConfig struct {
	// Host is the InfluxDB server URL/host.
	// +optional
	Host *string `json:"host,omitempty"`
	// Token is the InfluxDB API token. Use token_ref to reference a Secret instead.
	// +optional
	Token *string `json:"token,omitempty"`
	// TokenRef references a Secret containing the InfluxDB API token.
	// +optional
	TokenRef *corev1.SecretKeySelector `json:"token_ref,omitempty"`
	// Org is the organization name.
	// +optional
	Org *string `json:"org,omitempty"`
	// OrgID is the organization ID.
	// +optional
	OrgID *string `json:"org_id,omitempty"`
	// Bucket is the bucket name.
	// +optional
	Bucket *string `json:"bucket,omitempty"`
	// BucketID is the bucket ID.
	// +optional
	BucketID *string `json:"bucket_id,omitempty"`
	// SkipVerify skips TLS certificate verification.
	// +optional
	SkipVerify *bool `json:"skip_verify,omitempty"`
	// HTTPDebug enables HTTP debug logging.
	// +optional
	HTTPDebug *bool `json:"http_debug,omitempty"`
	// AllDatabases backs up all buckets.
	// +optional
	AllDatabases *bool `json:"all_databases,omitempty"`
}

// EtcdConfig holds the gobackup keys for the etcd backend.
type EtcdConfig struct {
	// Endpoint is a single etcd endpoint.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`
	// Endpoints are the etcd endpoints.
	// +optional
	Endpoints []string `json:"endpoints,omitempty"`
	// Args are additional etcdctl arguments.
	// +optional
	Args *string `json:"args,omitempty"`
}

// FirebirdConfig holds the gobackup keys for the firebird backend.
type FirebirdConfig struct {
	// Host is the Firebird server hostname.
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the Firebird server port.
	// +optional
	Port *int `json:"port,omitempty"`
	// Database is the database path/name.
	// +optional
	Database *string `json:"database,omitempty"`
	// Username is the database username. Use username_ref to reference a Secret instead.
	// +optional
	Username *string `json:"username,omitempty"`
	// UsernameRef references a Secret containing the database username.
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"username_ref,omitempty"`
	// Password is the database password. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the database password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Role is the SQL role to assume.
	// +optional
	Role *string `json:"role,omitempty"`
	// Args are additional gbak arguments.
	// +optional
	Args *string `json:"args,omitempty"`
}

// SQLiteConfig holds the gobackup keys for the sqlite backend.
type SQLiteConfig struct {
	// Path is the path to the SQLite database file.
	// +optional
	Path *string `json:"path,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// ObservedGeneration is the most recent Database spec generation observed by
	// the controller. There is no dedicated Database controller yet, so this is
	// currently unpopulated; it exists to make the status subresource honest.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:resource:shortName=db,categories=gobackup
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
