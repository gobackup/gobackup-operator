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
	// Type is the storage backend type (s3, gcs, azure, local, ftp, etc.) matching the Storage resource's spec.type field
	Type    string `json:"type,omitempty"`
	Name    string `json:"name,omitempty"`
	Keep    int    `json:"keep,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type DatabaseRef struct {
	APIGroup string `json:"apiGroup,omitempty"`
	// Type is the database backend type (postgresql, redis, etc.) matching the Database resource's spec.type field
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type Compress struct {
	Type string `json:"type,omitempty"`
}

type Encode struct {
	Type string `json:"type,omitempty"`
}

// BackupRunStatus represents the status of a single backup run
type BackupRunStatus struct {
	// JobName is the name of the Job that ran this backup
	JobName string `json:"jobName,omitempty"`

	// StartTime is when the backup job started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the backup job completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Phase is the current phase of the backup (Pending, Running, Succeeded, Failed)
	Phase string `json:"phase,omitempty"`

	// Message contains a human-readable message indicating details about the backup
	// This is truncated to avoid status size issues (max 1024 characters)
	Message string `json:"message,omitempty"`

	// Logs contains the last N lines of gobackup output (truncated to avoid large status)
	// Only captured on failure to help debugging. Max 4096 characters.
	// +optional
	Logs string `json:"logs,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	// LastBackupTime is the timestamp of the last backup attempt
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// LastSuccessfulBackupTime is the timestamp of the last successful backup
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`

	// Phase is the current phase of the backup (Idle, Running, Succeeded, Failed)
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the backup's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastRun contains the status of the most recent backup run
	// Only the last run is kept to avoid status size explosion
	// +optional
	LastRun *BackupRunStatus `json:"lastRun,omitempty"`

	// RecentRuns contains the status of recent backup runs (limited to last N runs)
	// This is a sliding window - oldest entries are removed when limit is exceeded
	// Default limit: 5 runs to prevent status bloat
	// +optional
	// +kubebuilder:validation:MaxItems=5
	RecentRuns []BackupRunStatus `json:"recentRuns,omitempty"`

	// FailureCount tracks consecutive failures for alerting purposes
	FailureCount int32 `json:"failureCount,omitempty"`

	// SuccessCount tracks total successful backups
	SuccessCount int32 `json:"successCount,omitempty"`
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
