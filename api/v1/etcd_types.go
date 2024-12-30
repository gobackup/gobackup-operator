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

// ETCDSpec defines the desired state of ETCD
type ETCDSpec struct {
	Endpoints         []string `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	AdditionalOptions string   `json:"additionalOptions,omitempty" yaml:"additionalOptions,omitempty"`
}

// ETCDSpecConfig duplicates ETCDSpec for gobackup config file
type ETCDSpecConfig struct {
	Endpoints         []string `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	AdditionalOptions string   `json:"additionalOptions,omitempty" yaml:"additionalOptions,omitempty"`
}

// ETCDStatus defines the observed state of ETCD
type ETCDStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ETCD is the Schema for the etcds API
type ETCD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ETCDSpec   `json:"spec,omitempty"`
	Status ETCDStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ETCDList contains a list of ETCD
type ETCDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ETCD `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ETCD{}, &ETCDList{})
}
