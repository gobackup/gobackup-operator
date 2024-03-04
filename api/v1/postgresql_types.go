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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgreSQLSpec defines the desired state of PostgreSQL
type PostgreSQLSpec struct {
	Host              string   `json:"host,omitempty" yaml:"host,omitempty"`
	Port              int      `json:"port,omitempty" yaml:"port,omitempty"`
	Type              string   `json:"type,omitempty" yaml:"type,omitempty"`
	Database          string   `json:"database,omitempty" yaml:"database,omitempty"`
	Username          string   `json:"username,omitempty" yaml:"username,omitempty"`
	Password          string   `json:"password,omitempty" yaml:"password,omitempty"`
	Tables            []string `json:"tables,omitempty" yaml:"tables,omitempty"`
	ExcludeTables     []string `json:"excludeTables,omitempty" yaml:"excludeTables,omitempty"`
	AdditionalOptions string   `json:"additionalOptions,omitempty" yaml:"additionalOptions,omitempty"`
}

// PostgreSQLSpec duplicates PostgreSQL for gobackup config file
type PostgreSQLSpecConfig struct {
	Host              string   `json:"host,omitempty" yaml:"host,omitempty"`
	Port              int      `json:"port,omitempty" yaml:"port,omitempty"`
	Type              string   `json:"type,omitempty" yaml:"type,omitempty"`
	Database          string   `json:"database,omitempty" yaml:"database,omitempty"`
	Username          string   `json:"username,omitempty" yaml:"username,omitempty"`
	Password          string   `json:"password,omitempty" yaml:"password,omitempty"`
	Tables            []string `json:"tables,omitempty" yaml:"tables,omitempty"`
	ExcludeTables     []string `json:"exclude_tables,omitempty" yaml:"exclude_tables,omitempty"`
	AdditionalOptions string   `json:"additional_options,omitempty" yaml:"additional_options,omitempty"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
type PostgreSQLStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgreSQL is the Schema for the postgresqls API
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec   `json:"spec,omitempty"`
	Status PostgreSQLStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
