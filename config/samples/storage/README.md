# Storage CRD Samples

This directory contains sample manifests for the unified Storage CRD that supports all gobackup storage backends.

## Directory Structure

```
samples/
├── secrets/               # Example Secrets for storing credentials
│   └── storage-credentials.yaml
├── storage/              # Individual storage type examples
│   ├── local.yaml        # Local filesystem storage
│   ├── s3-direct.yaml    # AWS S3 with inline credentials
│   ├── s3-secrets.yaml   # AWS S3 using SecretRef pattern
│   ├── gcs.yaml          # Google Cloud Storage
│   ├── azure.yaml        # Azure Blob Storage
│   ├── ftp.yaml          # FTP server
│   ├── sftp.yaml         # SFTP server
│   ├── scp.yaml          # SCP (SSH)
│   ├── webdav.yaml       # WebDAV server
│   └── minio.yaml        # MinIO object storage
└── complete/             # End-to-end examples
    └── test-backup-with-storage.yaml
```

## Storage Types Supported

The unified Storage CRD supports 18 storage backends:

### File Transfer Protocols
- **local** - Local filesystem storage
- **ftp** - FTP server
- **sftp** - SFTP (SSH File Transfer Protocol)
- **scp** - SCP (Secure Copy Protocol)
- **webdav** - WebDAV server

### Cloud Object Storage
- **s3** - Amazon AWS S3
- **gcs** - Google Cloud Storage
- **azure** - Azure Blob Storage

### S3-Compatible Services
- **oss** - Aliyun OSS
- **r2** - Cloudflare R2
- **spaces** - DigitalOcean Spaces
- **b2** - Backblaze B2
- **minio** - MinIO
- **cos** - QCloud COS (Tencent Cloud)
- **us3** - UCloud US3
- **kodo** - Qiniu Kodo
- **bos** - Baidu BOS
- **obs** - Huawei OBS
- **tos** - Volcengine TOS
- **upyun** - UpYun

## Using SecretRef Pattern

For better security, use Kubernetes Secrets to store sensitive credentials instead of inline values:

### Step 1: Create a Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: storage-credentials
  namespace: default
type: Opaque
stringData:
  s3-access-key-id: "YOUR_ACCESS_KEY"
  s3-secret-access-key: "YOUR_SECRET_KEY"
```

### Step 2: Reference the Secret in Storage

```yaml
apiVersion: gobackup.io/v1
kind: Storage
metadata:
  name: my-s3-storage
spec:
  type: s3
  config:
    bucket: my-bucket
    access_key_id_ref:
      name: storage-credentials
      key: s3-access-key-id
    secret_access_key_ref:
      name: storage-credentials
      key: s3-secret-access-key
```

## SecretRef Fields

The following fields support SecretRef pattern (use `field_ref` instead of `field`):

- `password` / `password_ref` - For FTP, SFTP, SCP, WebDAV
- `private_key` / `private_key_ref` - For SFTP, SCP
- `passphrase` / `passphrase_ref` - For SFTP, SCP (SSH key passphrase)
- `access_key_id` / `access_key_id_ref` - For S3-compatible services
- `secret_access_key` / `secret_access_key_ref` - For S3-compatible services
- `credentials` / `credentials_ref` - For GCS
- `client_secret` / `client_secret_ref` - For Azure

**Note:** When both direct value and SecretRef are provided, SecretRef takes precedence.

## Required Fields by Storage Type

### local
- `path` (required) - Local directory path

### ftp, sftp, scp
- `host` (required) - Server hostname
- Authentication (username + password OR private_key)

### webdav
- `root` (required) - WebDAV server root URL

### s3, oss, gcs, r2, spaces, b2, cos, us3, kodo, bos, minio, obs, tos, upyun
- `bucket` (required) - Bucket name
- Credentials (access keys or service account)

### azure
- `account` (required) - Azure Storage Account name
- `container` (required) - Container name
- `tenant_id`, `client_id`, `client_secret` (required) - Azure authentication

## Default Values

Fields have sensible defaults from gobackup:

- `timeout` - 300 seconds
- `max_retries` - 3
- `region` - Varies by provider (s3: us-east-1, oss: cn-hangzhou, etc.)
- `port` - Protocol-specific (ftp: 21, sftp/scp: 22)
- `storage_class` - Provider-specific (s3: STANDARD_IA)

## Complete Example

See `complete/test-backup-with-storage.yaml` for a full end-to-end example that includes:
- Secret with credentials
- Storage resource using SecretRef
- Database resource
- Backup resource with schedule

## Usage in Backup Resource

Reference storage in a Backup using the storage backend type:

```yaml
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: my-backup
spec:
  storageRefs:
    - apiGroup: gobackup.io
      type: s3              # Storage backend type
      name: my-s3-storage   # Storage resource name
      keep: 30
      timeout: 600
```

The `type` field should match the `spec.type` of the Storage resource being referenced.
