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

// MySQLSpec defines the desired state of MySQL
type MySQLSpec struct {
	Host              string   `json:"host,omitempty" yaml:"host,omitempty"`
	Port              int      `json:"port,omitempty" yaml:"port,omitempty"`
	Username          string   `json:"username,omitempty" yaml:"username,omitempty"`
	Password          string   `json:"password,omitempty" yaml:"password,omitempty"`
	Database          string   `json:"database,omitempty" yaml:"database,omitempty"`
	Tables            []string `json:"tables,omitempty" yaml:"tables,omitempty"`
	ExcludeTables     []string `json:"excludeTables,omitempty" yaml:"excludeTables,omitempty"`
	AdditionalOptions string   `json:"additionalOptions,omitempty" yaml:"additionalOptions,omitempty"`
}

// MySQLSpecConfig duplicates MySQLSpec for gobackup config file
type MySQLSpecConfig struct {
	Host              string   `json:"host,omitempty" yaml:"host,omitempty"`
	Port              int      `json:"port,omitempty" yaml:"port,omitempty"`
	Username          string   `json:"username,omitempty" yaml:"username,omitempty"`
	Password          string   `json:"password,omitempty" yaml:"password,omitempty"`
	Database          string   `json:"database,omitempty" yaml:"database,omitempty"`
	Tables            []string `json:"tables,omitempty" yaml:"tables,omitempty"`
	ExcludeTables     []string `json:"excludeTables,omitempty" yaml:"excludeTables,omitempty"`
	AdditionalOptions string   `json:"additionalOptions,omitempty" yaml:"additionalOptions,omitempty"`
}

// MySQLStatus defines the observed state of MySQL
type MySQLStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MySQL is the Schema for the mysqls API
type MySQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MySQLSpec   `json:"spec,omitempty"`
	Status MySQLStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MySQLList contains a list of MySQL
type MySQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MySQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MySQL{}, &MySQLList{})
}
