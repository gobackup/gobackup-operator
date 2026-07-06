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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSpec defines the desired state of Storage.
//
// Config carries one nested sub-object per storage type; exactly the sub-object
// matching spec.type must be set. The two CEL rules below enforce that invariant
// (the emitter flattens the chosen sub-object into gobackup.yml). The s3-family
// backends (s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos,
// upyun) all share S3CompatibleConfig; sftp and scp share SSHConfig.
//
// +kubebuilder:validation:XValidation:rule="(has(self.config.local)?1:0)+(has(self.config.s3)?1:0)+(has(self.config.oss)?1:0)+(has(self.config.r2)?1:0)+(has(self.config.spaces)?1:0)+(has(self.config.b2)?1:0)+(has(self.config.cos)?1:0)+(has(self.config.us3)?1:0)+(has(self.config.kodo)?1:0)+(has(self.config.bos)?1:0)+(has(self.config.minio)?1:0)+(has(self.config.obs)?1:0)+(has(self.config.tos)?1:0)+(has(self.config.upyun)?1:0)+(has(self.config.gcs)?1:0)+(has(self.config.azure)?1:0)+(has(self.config.ftp)?1:0)+(has(self.config.sftp)?1:0)+(has(self.config.scp)?1:0)+(has(self.config.webdav)?1:0) == 1",message="exactly one config sub-object must be set"
// +kubebuilder:validation:XValidation:rule="(self.type == 'local' && has(self.config.local)) || (self.type == 's3' && has(self.config.s3)) || (self.type == 'oss' && has(self.config.oss)) || (self.type == 'r2' && has(self.config.r2)) || (self.type == 'spaces' && has(self.config.spaces)) || (self.type == 'b2' && has(self.config.b2)) || (self.type == 'cos' && has(self.config.cos)) || (self.type == 'us3' && has(self.config.us3)) || (self.type == 'kodo' && has(self.config.kodo)) || (self.type == 'bos' && has(self.config.bos)) || (self.type == 'minio' && has(self.config.minio)) || (self.type == 'obs' && has(self.config.obs)) || (self.type == 'tos' && has(self.config.tos)) || (self.type == 'upyun' && has(self.config.upyun)) || (self.type == 'gcs' && has(self.config.gcs)) || (self.type == 'azure' && has(self.config.azure)) || (self.type == 'ftp' && has(self.config.ftp)) || (self.type == 'sftp' && has(self.config.sftp)) || (self.type == 'scp' && has(self.config.scp)) || (self.type == 'webdav' && has(self.config.webdav))",message="the config sub-object must match spec.type"
type StorageSpec struct {
	// Type is the storage backend type.
	// +kubebuilder:validation:Enum=local;ftp;sftp;scp;webdav;s3;oss;gcs;azure;r2;spaces;b2;cos;us3;kodo;bos;minio;obs;tos;upyun
	Type string `json:"type"`

	// Config carries exactly one nested sub-object matching spec.type.
	Config StorageConfig `json:"config"`
}

// StorageConfig holds one nested configuration sub-object per storage type.
// Exactly the sub-object matching StorageSpec.Type is set; the others are nil.
type StorageConfig struct {
	// Local filesystem storage (spec.type: local).
	// +optional
	Local *LocalConfig `json:"local,omitempty"`
	// Amazon S3 storage (spec.type: s3).
	// +optional
	S3 *S3CompatibleConfig `json:"s3,omitempty"`
	// Aliyun OSS storage (spec.type: oss).
	// +optional
	OSS *S3CompatibleConfig `json:"oss,omitempty"`
	// Cloudflare R2 storage (spec.type: r2).
	// +optional
	R2 *S3CompatibleConfig `json:"r2,omitempty"`
	// DigitalOcean Spaces storage (spec.type: spaces).
	// +optional
	Spaces *S3CompatibleConfig `json:"spaces,omitempty"`
	// Backblaze B2 storage (spec.type: b2).
	// +optional
	B2 *S3CompatibleConfig `json:"b2,omitempty"`
	// Tencent COS storage (spec.type: cos).
	// +optional
	COS *S3CompatibleConfig `json:"cos,omitempty"`
	// UCloud US3 storage (spec.type: us3).
	// +optional
	US3 *S3CompatibleConfig `json:"us3,omitempty"`
	// Qiniu Kodo storage (spec.type: kodo).
	// +optional
	Kodo *S3CompatibleConfig `json:"kodo,omitempty"`
	// Baidu BOS storage (spec.type: bos).
	// +optional
	BOS *S3CompatibleConfig `json:"bos,omitempty"`
	// MinIO storage (spec.type: minio).
	// +optional
	MinIO *S3CompatibleConfig `json:"minio,omitempty"`
	// Huawei OBS storage (spec.type: obs).
	// +optional
	OBS *S3CompatibleConfig `json:"obs,omitempty"`
	// Volcengine TOS storage (spec.type: tos).
	// +optional
	TOS *S3CompatibleConfig `json:"tos,omitempty"`
	// UpYun storage (spec.type: upyun).
	// +optional
	UpYun *S3CompatibleConfig `json:"upyun,omitempty"`
	// Google Cloud Storage (spec.type: gcs).
	// +optional
	GCS *GCSConfig `json:"gcs,omitempty"`
	// Azure Blob Storage (spec.type: azure).
	// +optional
	Azure *AzureConfig `json:"azure,omitempty"`
	// FTP storage (spec.type: ftp).
	// +optional
	FTP *FTPConfig `json:"ftp,omitempty"`
	// SFTP storage (spec.type: sftp).
	// +optional
	SFTP *SSHConfig `json:"sftp,omitempty"`
	// SCP storage (spec.type: scp).
	// +optional
	SCP *SSHConfig `json:"scp,omitempty"`
	// WebDAV storage (spec.type: webdav).
	// +optional
	WebDAV *WebDAVConfig `json:"webdav,omitempty"`
}

// LocalConfig holds the gobackup keys for the local backend.
type LocalConfig struct {
	// Path is the local directory backups are written to.
	// +optional
	Path *string `json:"path,omitempty"`
}

// S3CompatibleConfig holds the gobackup keys shared by the S3-family backends
// (s3, oss, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun).
type S3CompatibleConfig struct {
	// Bucket is the bucket name.
	// +optional
	Bucket *string `json:"bucket,omitempty"`
	// Region is the storage region.
	// +optional
	Region *string `json:"region,omitempty"`
	// Endpoint is the custom endpoint URL.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`
	// Path is the remote path prefix for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// AccessKeyID is the access key ID. Use access_key_id_ref to reference a Secret instead.
	// +optional
	AccessKeyID *string `json:"access_key_id,omitempty"`
	// AccessKeyIDRef references a Secret containing the access key ID.
	// +optional
	AccessKeyIDRef *corev1.SecretKeySelector `json:"access_key_id_ref,omitempty"`
	// SecretAccessKey is the secret access key. Use secret_access_key_ref to reference a Secret instead.
	// +optional
	SecretAccessKey *string `json:"secret_access_key,omitempty"`
	// SecretAccessKeyRef references a Secret containing the secret access key.
	// +optional
	SecretAccessKeyRef *corev1.SecretKeySelector `json:"secret_access_key_ref,omitempty"`
	// Token is the session token for temporary credentials.
	// +optional
	Token *string `json:"token,omitempty"`
	// StorageClass is the object storage class.
	// +optional
	StorageClass *string `json:"storage_class,omitempty"`
	// MaxRetries is the maximum number of retry attempts.
	// +optional
	MaxRetries *int `json:"max_retries,omitempty"`
	// ForcePathStyle forces path-style URLs instead of virtual-hosted-style.
	// +optional
	ForcePathStyle *bool `json:"force_path_style,omitempty"`
	// AccountID is the account identifier (e.g. Cloudflare R2 account ID).
	// +optional
	AccountID *string `json:"account_id,omitempty"`
	// Timeout is the upload timeout in seconds.
	// +kubebuilder:default=300
	// +optional
	Timeout *int `json:"timeout,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// GCSConfig holds the gobackup keys for the gcs backend.
type GCSConfig struct {
	// Bucket is the GCS bucket name.
	// +optional
	Bucket *string `json:"bucket,omitempty"`
	// Path is the remote path prefix for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// Credentials is the JSON content of the Google Cloud credentials. Use credentials_ref to reference a Secret instead.
	// +optional
	Credentials *string `json:"credentials,omitempty"`
	// CredentialsRef references a Secret containing the Google Cloud credentials JSON.
	// +optional
	CredentialsRef *corev1.SecretKeySelector `json:"credentials_ref,omitempty"`
	// CredentialsFile is the path to a Google Cloud credentials file.
	// +optional
	CredentialsFile *string `json:"credentials_file,omitempty"`
	// Timeout is the upload timeout in seconds.
	// +kubebuilder:default=300
	// +optional
	Timeout *int `json:"timeout,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// AzureConfig holds the gobackup keys for the azure backend.
type AzureConfig struct {
	// Account is the Azure Storage Account name.
	// +optional
	Account *string `json:"account,omitempty"`
	// Container is the Azure container name.
	// +optional
	Container *string `json:"container,omitempty"`
	// Bucket is an alias for the Azure Storage Account name.
	// +optional
	Bucket *string `json:"bucket,omitempty"`
	// Path is the remote path prefix for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// ClientID is the Azure Client ID.
	// +optional
	ClientID *string `json:"client_id,omitempty"`
	// ClientSecret is the Azure Client Secret. Use client_secret_ref to reference a Secret instead.
	// +optional
	ClientSecret *string `json:"client_secret,omitempty"`
	// ClientSecretRef references a Secret containing the Azure Client Secret.
	// +optional
	ClientSecretRef *corev1.SecretKeySelector `json:"client_secret_ref,omitempty"`
	// TenantID is the Azure Tenant ID.
	// +optional
	TenantID *string `json:"tenant_id,omitempty"`
	// Timeout is the upload timeout in seconds.
	// +kubebuilder:default=300
	// +optional
	Timeout *int `json:"timeout,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// FTPConfig holds the gobackup keys for the ftp backend.
type FTPConfig struct {
	// Host is the FTP server hostname.
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the FTP server port.
	// +optional
	Port *int `json:"port,omitempty"`
	// Path is the remote path for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// Username for authentication.
	// +optional
	Username *string `json:"username,omitempty"`
	// Password for authentication. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// TLS enables implicit FTPS.
	// +optional
	TLS *bool `json:"tls,omitempty"`
	// ExplicitTLS enables explicit FTPS.
	// +optional
	ExplicitTLS *bool `json:"explicit_tls,omitempty"`
	// NoCheckCertificate skips TLS certificate verification.
	// +optional
	NoCheckCertificate *bool `json:"no_check_certificate,omitempty"`
	// Timeout is the upload timeout in seconds.
	// +kubebuilder:default=300
	// +optional
	Timeout *int `json:"timeout,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// SSHConfig holds the gobackup keys shared by the sftp and scp backends.
type SSHConfig struct {
	// Host is the server hostname.
	// +optional
	Host *string `json:"host,omitempty"`
	// Port is the server port.
	// +optional
	Port *int `json:"port,omitempty"`
	// Path is the remote path for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// Username for authentication.
	// +optional
	Username *string `json:"username,omitempty"`
	// Password for authentication. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// PrivateKey is the path to the SSH private key.
	// +optional
	PrivateKey *string `json:"private_key,omitempty"`
	// PrivateKeyRef references a Secret containing the SSH private key content.
	// +optional
	PrivateKeyRef *corev1.SecretKeySelector `json:"private_key_ref,omitempty"`
	// Passphrase is the password for the private key if present.
	// +optional
	Passphrase *string `json:"passphrase,omitempty"`
	// PassphraseRef references a Secret containing the private key passphrase.
	// +optional
	PassphraseRef *corev1.SecretKeySelector `json:"passphrase_ref,omitempty"`
	// Timeout is the upload timeout in seconds.
	// +kubebuilder:default=300
	// +optional
	Timeout *int `json:"timeout,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// WebDAVConfig holds the gobackup keys for the webdav backend.
type WebDAVConfig struct {
	// Path is the remote path for backups.
	// +optional
	Path *string `json:"path,omitempty"`
	// Root is the WebDAV server root URL.
	// +optional
	Root *string `json:"root,omitempty"`
	// Username for authentication.
	// +optional
	Username *string `json:"username,omitempty"`
	// Password for authentication. Use password_ref to reference a Secret instead.
	// +optional
	Password *string `json:"password,omitempty"`
	// PasswordRef references a Secret containing the password.
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"password_ref,omitempty"`
	// Keep specifies how many backups to retain at this storage location.
	// +optional
	Keep *int `json:"keep,omitempty"`
}

// StorageStatus defines the observed state of Storage
type StorageStatus struct {
	// ObservedGeneration is the most recent Storage spec generation observed by
	// the controller. There is no dedicated Storage controller yet, so this is
	// currently unpopulated; it exists to make the status subresource honest.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:resource:shortName=storage,categories=gobackup
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
