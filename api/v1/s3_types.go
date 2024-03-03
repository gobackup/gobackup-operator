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

// S3Spec defines the desired state of S3
type S3Spec struct {
	Bucket          string `json:"bucket" yaml:"bucket"`
	Region          string `json:"region" yaml:"region"`
	AccessKeyID     string `json:"accessKeyID" yaml:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey" yaml:"secretAccessKey"`
	MaxRetries      int    `json:"maxRetries" yaml:"maxRetries"`
	StorageClass    string `json:"storageClass" yaml:"storageClass"`
	ForcePathStyle  bool   `json:"forcePathStyle" yaml:"forcePathStyle"`
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
}

// S3Status defines the observed state of S3
type S3Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// S3 is the Schema for the s3s API
type S3 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   S3Spec   `json:"spec,omitempty"`
	Status S3Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// S3List contains a list of S3
type S3List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []S3 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&S3{}, &S3List{})
}
