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

// BackupModelSpec defines the desired state of BackupModel
type BackupModelSpec struct {
	Description  string   `json:"description"`
	CompressWith Compress `json:"compress_with"`
	EncodeWith   Encode   `json:"encode_with"`
	BeforeScript string   `json:"before_script"`
	AfterScript  string   `json:"after_script"`
}

// BackupModelStatus defines the observed state of BackupModel
type BackupModelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type Compress struct {
	Type string `json:"type"`
}

type Encode struct {
	Openssl  bool   `json:"openssl"`
	Salt     bool   `json:"salt"`
	Base64   bool   `json:"base64"`
	Password string `json:"password"`
	Args     string `json:"args"`
	Cipher   string `json:"cipher"`
	Type     string `json:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BackupModel is the Schema for the backupmodels API
type BackupModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupModelSpec   `json:"spec,omitempty"`
	Status BackupModelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupModelList contains a list of BackupModel
type BackupModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupModel{}, &BackupModelList{})
}
