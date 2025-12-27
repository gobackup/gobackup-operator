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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSpec defines the desired state of Storage
type StorageSpec struct {
	// Type is the storage backend type
	// +kubebuilder:validation:Enum=local;ftp;sftp;scp;webdav;s3;oss;gcs;azure;r2;spaces;b2;cos;us3;kodo;bos;minio;obs;tos;upyun
	Type string `json:"type"`

	// Config contains the storage configuration
	Config StorageConfig `json:"config"`
}

// StorageConfig defines the configuration for all storage types
type StorageConfig struct {
	// Common fields applicable to all or multiple storage types

	// Path is the remote path for saving backup files
	// Used by: all storage types except azure (uses container)
	Path *string `json:"path,omitempty"`

	// Timeout is the upload timeout in seconds. Default: 300
	// Used by: s3, oss, ftp, sftp, scp, gcs, azure, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	Timeout *int `json:"timeout,omitempty"`

	// Keep specifies how many backups to retain at this storage location
	// Used by: all storage types
	Keep *int `json:"keep,omitempty"`

	// Authentication fields

	// Username for authentication
	// Used by: ftp, sftp, scp, webdav
	Username *string `json:"username,omitempty"`

	// Password for authentication. Use password_ref to reference a Secret instead
	// Used by: ftp, sftp, scp, webdav
	Password *string `json:"password,omitempty"`

	// PasswordRef references a Secret containing the password
	// Used by: ftp, sftp, scp, webdav
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`

	// PrivateKey is the path to SSH private key. Default: ~/.ssh/id_rsa
	// Used by: sftp, scp
	PrivateKey *string `json:"private_key,omitempty"`

	// PrivateKeyRef references a Secret containing the SSH private key content
	// Used by: sftp, scp
	PrivateKeyRef *corev1.SecretKeySelector `json:"private_key_ref,omitempty"`

	// Passphrase is the password for the private key if present
	// Used by: sftp, scp
	Passphrase *string `json:"passphrase,omitempty"`

	// PassphraseRef references a Secret containing the private key passphrase
	// Used by: sftp, scp
	PassphraseRef *corev1.SecretKeySelector `json:"passphrase_ref,omitempty"`

	// S3-compatible storage fields (s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun)

	// Bucket is the bucket/container name
	// Required for: s3, oss, gcs, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	Bucket *string `json:"bucket,omitempty"`

	// Region is the storage region. Default varies by provider (s3: us-east-1, oss: cn-hangzhou, spaces: nyc1, b2: us-east-001, minio: us-east-1)
	// Used by: s3, oss, spaces, b2, minio, and other S3-compatible services
	Region *string `json:"region,omitempty"`

	// Endpoint is the custom endpoint URL for S3-compatible services
	// Used by: s3, oss, r2, spaces, b2, minio, and other S3-compatible services
	Endpoint *string `json:"endpoint,omitempty"`

	// AccessKeyID is the access key ID. Use access_key_id_ref to reference a Secret instead
	// Used by: s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	AccessKeyID *string `json:"access_key_id,omitempty"`

	// AccessKeyIDRef references a Secret containing the access key ID
	// Used by: s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	AccessKeyIDRef *corev1.SecretKeySelector `json:"access_key_id_ref,omitempty"`

	// SecretAccessKey is the secret access key. Use secret_access_key_ref to reference a Secret instead
	// Used by: s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	SecretAccessKey *string `json:"secret_access_key,omitempty"`

	// SecretAccessKeyRef references a Secret containing the secret access key
	// Used by: s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	SecretAccessKeyRef *corev1.SecretKeySelector `json:"secret_access_key_ref,omitempty"`

	// StorageClass is the storage class. Default varies by provider (s3: STANDARD_IA, oss: STANDARD_IA, spaces: STANDARD, b2: STANDARD)
	// Used by: s3, oss, spaces, b2
	StorageClass *string `json:"storage_class,omitempty"`

	// MaxRetries is the maximum number of retry attempts. Default: 3
	// Used by: s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
	MaxRetries *int `json:"max_retries,omitempty"`

	// ForcePathStyle forces path-style URLs instead of virtual-hosted-style
	// Used by: s3 and S3-compatible services
	ForcePathStyle *bool `json:"force_path_style,omitempty"`

	// AccountID is the account identifier
	// Required for: r2 (Cloudflare R2)
	AccountID *string `json:"account_id,omitempty"`

	// Google Cloud Storage fields

	// Credentials is the JSON content of Google Cloud Application Credentials. Use credentials_ref to reference a Secret instead
	// Used by: gcs
	Credentials *string `json:"credentials,omitempty"`

	// CredentialsRef references a Secret containing the Google Cloud Application Credentials JSON
	// Used by: gcs
	CredentialsRef *corev1.SecretKeySelector `json:"credentials_ref,omitempty"`

	// CredentialsFile is the path to Google Cloud Application Credentials file
	// Used by: gcs
	CredentialsFile *string `json:"credentials_file,omitempty"`

	// Azure Blob Storage fields

	// Account is the Azure Storage Account name (alias: bucket)
	// Required for: azure
	Account *string `json:"account,omitempty"`

	// Container is the Azure container name
	// Required for: azure
	Container *string `json:"container,omitempty"`

	// TenantID is the Azure Tenant ID (format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	// Required for: azure
	TenantID *string `json:"tenant_id,omitempty"`

	// ClientID is the Azure Client ID (format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	// Required for: azure
	ClientID *string `json:"client_id,omitempty"`

	// ClientSecret is the Azure Client Secret. Use client_secret_ref to reference a Secret instead
	// Required for: azure
	ClientSecret *string `json:"client_secret,omitempty"`

	// ClientSecretRef references a Secret containing the Azure Client Secret
	// Used by: azure
	ClientSecretRef *corev1.SecretKeySelector `json:"client_secret_ref,omitempty"`

	// Protocol-specific fields (FTP, SFTP, SCP, WebDAV)

	// Host is the server hostname
	// Required for: ftp, sftp, scp
	Host *string `json:"host,omitempty"`

	// Port is the server port. Default varies by protocol (ftp: 21, sftp: 22, scp: 22)
	// Used by: ftp, sftp, scp
	Port *int `json:"port,omitempty"`

	// Root is the WebDAV server root URL (e.g., http://localhost:8080)
	// Required for: webdav
	Root *string `json:"root,omitempty"`
}

// StorageStatus defines the observed state of Storage
type StorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:resource:shortName=storage
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Storage is the Schema for the storages API
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec,omitempty"`
	Status StorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
