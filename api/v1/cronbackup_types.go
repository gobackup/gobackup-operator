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

// CronBackupSpec defines the desired state of CronBackup
type CronBackupSpec struct {
}

type Schedule struct {
	Cron string `json:"cron,omitempty"`
}

type BackupModelRef struct {
	Name     string   `json:"name,omitempty"`
	Schedule Schedule `json:"schedule,omitempty"`
}

type StorageRef struct {
	APIGroup string `json:"apiGroup,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Keep     int    `json:"keep,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

type DatabaseRef struct {
	APIGroup string `json:"apiGroup,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
}

// CronBackupStatus defines the observed state of CronBackup
type CronBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CronBackup is the Schema for the cronbackups API
type CronBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronBackupSpec   `json:"spec,omitempty"`
	Status CronBackupStatus `json:"status,omitempty"`

	Model
}

//+kubebuilder:object:root=true

// CronBackupList contains a list of CronBackup
type CronBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronBackup{}, &CronBackupList{})
}
