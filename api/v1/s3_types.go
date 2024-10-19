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
	Type            string `json:"type,omitempty" yaml:"type,omitempty"`
	Bucket          string `json:"bucket,omitempty" yaml:"bucket,omitempty"`
	Region          string `json:"region,omitempty" yaml:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Path            string `json:"path,omitempty" yaml:"path,omitempty"`
	AccessKeyID     string `json:"accessKeyID,omitempty" yaml:"accessKeyID,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`
	ForcePathStyle  bool   `json:"forcePathStyle,omitempty" yaml:"forcePathStyle,omitempty"`
	StorageClass    string `json:"storageClass" yaml:"storageClass"`
	MaxRetries      int    `json:"maxRetries,omitempty" yaml:"maxRetries,omitempty"`
	Keep            int    `json:"keep,omitempty" yaml:"keep,omitempty"`
	Timeout         int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// S3SpecConfig duplicates S3Spec for gobackup config file
type S3SpecConfig struct {
	Type            string `json:"type,omitempty" yaml:"type,omitempty"`
	Bucket          string `json:"bucket,omitempty" yaml:"bucket,omitempty"`
	Region          string `json:"region,omitempty" yaml:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Path            string `json:"path,omitempty" yaml:"path,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty"`
	ForcePathStyle  bool   `json:"force_path_style,omitempty" yaml:"force_path_style,omitempty"`
	StorageClass    string `json:"storage_class" yaml:"storage_class"`
	MaxRetries      int    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	Keep            int    `json:"keep,omitempty" yaml:"keep,omitempty"`
	Timeout         int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// S3Status defines the observed state of S3
type S3Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:resource:shortName=s3
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
