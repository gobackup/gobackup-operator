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

### CustomResourceDefinitions

The Operator acts on the following [Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):

- `Backup`, which defines a backup operation configuration. It references one or more database resources and storage backends, and can be configured for immediate execution or scheduled backups using cron syntax. Supports compression, retention policies, and pre/post backup scripts.

#### Database

- `PostgreSQL`, which defines a PostgreSQL database connection configuration. It specifies connection details such as host, port, username, password, database name, and optional table inclusion/exclusion filters.
- `MySQL`, which defines a MySQL database connection configuration. It specifies connection details such as host, port, username, password, database name, and optional table inclusion/exclusion filters.
- `MariaDB`, which defines a MariaDB database connection configuration. It specifies connection details such as host, port, username, password, and database name.
- `MongoDB`, which defines a MongoDB database connection configuration. It specifies connection details such as host, port, username, password, database name, authentication database, and optional oplog backup settings.
- `Redis`, which defines a Redis database connection configuration. It specifies connection details such as host, port, and optional password.
- `MSSQL`, which defines a Microsoft SQL Server database connection configuration. It specifies connection details such as host, port, username, password, database name, and optional trust server certificate settings.
- `InfluxDB`, which defines an InfluxDB database connection configuration. It specifies connection details such as host, token, bucket, organization, and optional verification settings.
- `ETCD`, which defines an etcd cluster connection configuration. It specifies endpoints and optional additional backup options.

#### Storage

- `S3`, which defines an S3-compatible storage backend configuration. It specifies bucket, region, credentials, path, and other S3-specific settings for storing backups.
- `Azure`, which defines an Azure Blob Storage backend configuration. It specifies account, container, tenant ID, client ID, and client secret for authentication.
- `GCS`, which defines a Google Cloud Storage backend configuration. It specifies bucket, path, and credentials (either directly or via a secret reference).
- `WebDAV`, which defines a WebDAV storage backend configuration. It specifies root URL, username, and password for authentication.
- `FTP`, which defines an FTP storage backend configuration. It specifies host, port, username, password, path, and optional TLS settings.
- `SFTP`, which defines an SFTP storage backend configuration. It specifies host, port, username, and authentication via either password or private key with passphrase.
- `SCP`, which defines an SCP storage backend configuration. It specifies host, port, username, and authentication via either password or private key with passphrase.

### Installation

#### Option 1: Using Helm (Recommended)

Add the Helm repository and install the chart:

```sh
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace
```

Or install with custom values:

```sh
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace \
  --set image.tag=v0.1.0 \
  --set resources.limits.memory=256Mi
```

To upgrade:

```sh
helm upgrade gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system
```

To uninstall:

```sh
helm uninstall gobackup-operator --namespace gobackup-operator-system
```

See [charts/gobackup-operator/README.md](charts/gobackup-operator/README.md) for all available configuration options.

#### Option 2: Using Kustomize/Make

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

The operator follows best practices from well-known operators like prometheus-operator and ArgoCD operator, with comprehensive testing at multiple levels.

### Quick Test

A test script is provided to quickly test the operator functionality:

```sh
./hack/test-operator.sh
```

This script will:
1. Deploy the operator
2. Create example database and storage CRDs
3. Create both immediate and scheduled backup jobs
4. Display the status of the created resources

### Comprehensive Testing

The operator includes a three-tier testing strategy:

#### 1. Unit Tests

Fast unit tests that don't require a Kubernetes API server:

```sh
make test-unit
```

#### 2. Integration Tests

Integration tests using envtest (requires test binaries):

```sh
make test-integration
```

#### 3. End-to-End Tests

E2E tests that run against a real Kubernetes cluster:

```sh
# Using an existing cluster
make test-e2e

# Or using kind (local cluster)
make kind-run
make test-e2e
```

### Running All Tests

Run all tests including integration tests:

```sh
make test
```

### Test Coverage

Generate a test coverage report:

```sh
make test-coverage
```

This generates `cover.html` which you can open in a browser to see coverage details.

### Testing Best Practices

The test suite follows operator best practices:

- **Isolation**: Each test is independent and doesn't rely on other tests
- **Cleanup**: Resources are properly cleaned up after each test
- **Fixtures**: Helper functions create test resources consistently
- **Coverage**: Comprehensive test coverage for critical paths

For more detailed testing information, see [test/README.md](test/README.md).

## Structure

```
gobackup-operator/
├── .github/               # CI/CD workflows (GitHub Actions)
├── api/                   # API definitions (CustomResourceDefinitions)
├── build/                 # Build artifacts
├── charts/                # Helm charts
│   └── gobackup-operator/ # Main Helm chart for the operator
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
