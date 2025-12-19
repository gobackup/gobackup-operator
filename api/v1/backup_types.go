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

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// DatabaseRefs represents the list of databases to backup
	DatabaseRefs []DatabaseRef `json:"databaseRefs,omitempty"`

	// StorageRefs represents the list of storages to backup to
	StorageRefs []StorageRef `json:"storageRefs,omitempty"`

	// AfterScript is the script to run after the backup
	AfterScript string `json:"afterScript,omitempty"`

	// BeforeScript is the script to run before the backup
	BeforeScript string `json:"beforeScript,omitempty"`

	// CompressWith defines the compression to use
	CompressWith *Compress `json:"compressWith,omitempty"`

	// EncodeWith defines the encoding to use
	EncodeWith *Encode `json:"encodeWith,omitempty"`

	// Schedule defines when the backup should run
	Schedule *BackupSchedule `json:"schedule,omitempty"`

	// Persistence defines the storage for gobackup state (cycler.json)
	Persistence *Persistence `json:"persistence,omitempty"`
}

// Persistence defines the persistence configuration
type Persistence struct {
	// Enabled determines if a PVC should be created
	Enabled bool `json:"enabled,omitempty"`
	// StorageClass to use for the PVC
	StorageClass *string `json:"storageClass,omitempty"`
	// AccessMode for the PVC (default: ReadWriteOnce)
	AccessMode string `json:"accessMode,omitempty"`
	// Size of the PVC (default: 100Mi)
	Size string `json:"size,omitempty"`
}

// BackupSchedule defines the schedule for the backup
type BackupSchedule struct {
	// The cron expression defining the schedule
	Cron string `json:"cron,omitempty"`

	// Optional deadline in seconds for starting the job if it misses scheduled time for any reason
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// This flag tells the controller to suspend subsequent executions
	Suspend *bool `json:"suspend,omitempty"`

	// The number of successful finished jobs to retain
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// The number of failed finished jobs to retain
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
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

type Compress struct {
	Type string `json:"type,omitempty"`
}

type Encode struct {
	Type string `json:"type,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:resource:shortName=backup
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
