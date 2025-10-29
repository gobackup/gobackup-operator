<p align="center">
  <img src="https://github.com/user-attachments/assets/eb9f7270-9250-4d41-915b-c2debc873741" width="250" />
</p>

# gobackup-operator

**Please note:** This project is currently under active development.

A Kubernetes operator for backing up various databases to different storage providers using [gobackup](https://github.com/gobackup/gobackup).

## Description
GoBackup Operator allows you to define and manage backup operations for your databases in Kubernetes. 
It supports both immediate and scheduled backups, with configurable retention policies and compression options.

## Features

- Support for PostgreSQL databases (more coming soon)
- S3-compatible storage backends
- Scheduled backups with cron syntax
- One-time immediate backups
- Configurable retention policies (keep X most recent backups)
- Pre and post backup scripts
- Compression support

## Getting Started

### Prerequisites
- Kubernetes cluster
- kubectl
- make

### Installation

1. Install Custom Resource Definitions (CRDs):

```sh
make install
```

2. Deploy the operator:

```sh
make deploy
```

## Usage

### 1. Define your database

Create a PostgreSQL database reference:

```yaml
apiVersion: gobackup.io/v1
kind: PostgreSQL
metadata:
  name: my-postgres
  namespace: default
spec:
  host: "postgres.default.svc.cluster.local"
  port: 5432
  username: "postgres"
  password: "password"
  database: "my_database"
  # Optional: Specify tables to include
  # tables:
  #   - table1
  #   - table2
  # Optional: Specify tables to exclude
  # excludeTables:
  #   - logs
  #   - temp_data
  # Optional: Additional options for pg_dump
  # additionalOptions: "--no-owner --no-acl"
```

### 2. Define your storage backend

Create an S3 storage reference:

```yaml
apiVersion: gobackup.io/v1
kind: S3
metadata:
  name: my-s3
  namespace: default
spec:
  bucket: "my-backup-bucket"
  region: "us-east-1"
  # For S3-compatible services, specify the endpoint
  # endpoint: "minio.example.com"
  accessKeyID: "your-access-key"
  secretAccessKey: "your-secret-key"
  path: "backups"
  # Optional: Use path-style addressing instead of virtual-hosted style
  # forcePathStyle: true
  # Optional: Specify storage class (e.g., "STANDARD_IA", "GLACIER")
  # storageClass: "STANDARD"
  # Optional: Number of retry attempts
  # maxRetries: 3
```

### 3. Create a backup

#### Immediate (One-time) Backup

For an immediate, one-time backup:

```yaml
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: my-immediate-backup
  namespace: default
spec:
  databaseRefs:
    - apiGroup: gobackup.io
      type: PostgreSQL
      name: my-postgres
  storageRefs:
    - apiGroup: gobackup.io
      type: S3
      name: my-s3
      keep: 5  # Keep last 5 backups
      timeout: 300  # Timeout in seconds
  compressWith:
    type: gzip
  # Optional: Scripts to run before/after backup
  beforeScript: "echo 'Starting backup...'"
  afterScript: "echo 'Backup completed!'"
```

#### Scheduled Backup

For scheduled backups using cron syntax:

```yaml
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: my-scheduled-backup
  namespace: default
spec:
  databaseRefs:
    - apiGroup: gobackup.io
      type: PostgreSQL
      name: my-postgres
  storageRefs:
    - apiGroup: gobackup.io
      type: S3
      name: my-s3
      keep: 10  # Keep last 10 backups
      timeout: 300  # Timeout in seconds
  compressWith:
    type: gzip
  schedule:
    cron: "0 2 * * *"  # Run daily at 2am
    # Optional: Configure schedule behavior
    successfulJobsHistoryLimit: 3  # Keep history of last 3 successful jobs
    failedJobsHistoryLimit: 1  # Keep history of last failed job
    # suspend: true  # Set to true to temporarily pause the schedule
```

## Testing

A test script is provided to quickly test the operator functionality:

```sh
./hack/test-operator.sh
```

This script will:
1. Deploy the operator
2. Create example database and storage CRDs
3. Create both immediate and scheduled backup jobs
4. Display the status of the created resources

## Structure

```
gobackup-operator/
├── .github/               # CI/CD workflows (GitHub Actions)
├── api/                   # API definitions (CustomResourceDefinitions)
├── build/                 # Build artifacts
├── cmd/                   # Entry point for the operator
├── config/
│   ├── crd/               # Custom Resource Definitions (CRDs)
│   ├── default/           # Default manifests (e.g., manager deployment, cluster roles, RBAC)
│   ├── manager/           # Operator deployment manifests (e.g., Deployment.yaml, Service.yaml)
│   ├── rbac/              # RBAC permissions (e.g., ClusterRole.yaml, Role.yaml, RoleBinding.yaml)
│   ├── samples/           # Example custom resources (CRs) to test your operator
├── example.local/         # Example manifests for testing
├── internal/
│   ├──controller/         # Controller logic
├── pkg/                   # internal utils
├── Makefile               # Automation scripts (build, deploy, test)
├── PROJECT                # Operator SDK/Kubebuilder metadata
├── README.md              # Documentation
```

## Contributing

Just create a new branch (feature-{branch-name}) and push.

When you finish your work, please send a PR.

## License

MIT
