# External Integrations

**Analysis Date:** 2026-05-27

## GoBackup Tool (Core Integration)

**What it is:** `gobackup` is an external open-source backup binary (`github.com/huacnlee/gobackup`). The operator does NOT embed it or call it as a library; instead it spawns a Kubernetes Job/CronJob whose pod runs the `gobackup` container image and executes `gobackup perform`.

**Execution model:**
- Operator writes a `gobackup.yml` config file into a Kubernetes Secret (named after the Backup CR)
- The Job pod mounts the Secret at `/root/.gobackup/gobackup.yml` (the default config location for the gobackup binary)
- Pod command: `["/bin/sh", "-c", "gobackup perform"]`
- Image configured via `BACKUP_JOB_IMAGE` env var; Helm default: `ghcr.io/gobackup/gobackup:v3.1.0`

**Config generation:** `pkg/k8sutil/secret.go` — `CreateSecret()` translates CRD specs into YAML using `gopkg.in/yaml.v2` and the `BackupConfig` / `Model` structs.

**gobackup.yml structure generated:**
```yaml
models:
  <backup-name>:
    databases:
      <db-name>:
        type: postgresql   # or mysql, redis, mongodb, etc.
        host: ...
        # other db-specific fields
    storages:
      <storage-name>:
        type: s3           # or gcs, azure, minio, local, ftp, etc.
        bucket: ...
        # other storage-specific fields
    before_script: ...     # optional
    after_script: ...      # optional
    compress_with: gzip    # optional
    encode_with: ...       # optional
```

## Kubernetes API

**How accessed:** Two parallel clients initialized in `cmd/main.go`:

1. **controller-runtime generic client** (`sigs.k8s.io/controller-runtime/pkg/client`) — used inside `BackupReconciler` for all typed CRUD operations on CRDs, CronJobs, Jobs.
2. **Raw typed clientset** (`k8s.io/client-go/kubernetes`) via `pkg/k8sutil.NewClient()` — used in `pkg/k8sutil/secret.go` for Secret create/update/delete and in `backup_controller.go` for pod log streaming.
3. **Dynamic client** (`k8s.io/client-go/dynamic`) via `pkg/k8sutil.NewDynamicClient()` — used in `pkg/k8sutil/crd.go` `GetCRD()` to fetch arbitrary CRD instances (Database, Storage CRs) as `unstructured.Unstructured`.

**Auth:** In-cluster service account (auto-detected); falls back to kubeconfig for local development (`pkg/k8sutil/k8sutil.go` `getConfig()`).

**Resources managed via RBAC** (markers in `internal/controller/backup_controller.go`):

| Group | Resources | Verbs |
|-------|-----------|-------|
| `gobackup.io` | backups, databases, storages | get, list, watch, create, update, patch, delete |
| `gobackup.io` | backups/status, backups/finalizers | get, update, patch |
| `batch` | jobs, cronjobs | get, list, watch, create, update, patch, delete |
| `core` | secrets, pods | get, list, watch, create, update, patch, delete |
| `core` | pods/log | get |

## CRD-to-gobackup Config Pipeline

**Three CRDs defined in `api/v1/`:**

### Backup (`api/v1/backup_types.go`)
- Group/Version: `gobackup.io/v1`, Kind: `Backup`, shortName: `backup`
- References `Database` and `Storage` CRs by name
- Defines schedule (cron), compression, encryption, before/after scripts
- Status tracks: `LastBackupTime`, `LastSuccessfulBackupTime`, `Phase`, `RecentRuns` (max 5), `FailureCount`, `SuccessCount`, `ObservedGeneration`
- Controller: `internal/controller/backup_controller.go`

### Database (`api/v1/database_types.go`)
- Group/Version: `gobackup.io/v1`, Kind: `Database`, shortName: `db`
- `spec.type` enum: `postgresql`, `mysql`, `mariadb`, `mongodb`, `redis`, `mssql`, `influxdb`, `etcd`
- CEL validation rules enforce type-specific field constraints (e.g., `config.token` only valid for `influxdb`)
- Credential fields support both inline values and `SecretKeySelector` refs (e.g., `password_ref`, `username_ref`, `token_ref`)
- No dedicated controller; fetched dynamically by `BackupReconciler` via dynamic client when building config

### Storage (`api/v1/storage_types.go`)
- Group/Version: `gobackup.io/v1`, Kind: `Storage`, shortName: `storage`
- `spec.type` enum: `local`, `ftp`, `sftp`, `scp`, `webdav`, `s3`, `oss`, `gcs`, `azure`, `r2`, `spaces`, `b2`, `cos`, `us3`, `kodo`, `bos`, `minio`, `obs`, `tos`, `upyun`
- Credential fields support both inline values and `SecretKeySelector` refs (e.g., `access_key_id_ref`, `secret_access_key_ref`, `credentials_ref`, `client_secret_ref`, `private_key_ref`)
- No dedicated controller; fetched dynamically when building config

## Secret Reference Resolution

**Pattern:** Any CRD config field ending in `_ref` is treated as a `SecretKeySelector` reference. The operator resolves these at reconcile time by fetching the named Kubernetes Secret from the same namespace and substituting the plaintext value into the generated `gobackup.yml`.

**Implementation:** `pkg/k8sutil/secret.go` `resolveSecretReferences()` — iterates config maps, detects `_ref` suffix, fetches Secrets via raw clientset.

**Affected field patterns:**
- Database: `password_ref`, `username_ref`, `token_ref`
- Storage: `access_key_id_ref`, `secret_access_key_ref`, `credentials_ref`, `client_secret_ref`, `password_ref`, `private_key_ref`, `passphrase_ref`

## Storage Backend Providers

Configured entirely through Storage CRs — the operator has no direct SDK dependency on any cloud provider. All provider-specific auth and endpoint config is passed through to the `gobackup` binary via the generated YAML config.

**Supported providers (via `spec.type`):**

| Provider | Type value | Auth fields |
|----------|-----------|-------------|
| AWS S3 | `s3` | `access_key_id`, `secret_access_key` (or `_ref`) |
| Google Cloud Storage | `gcs` | `credentials` JSON or `credentials_ref` |
| Azure Blob Storage | `azure` | `tenant_id`, `client_id`, `client_secret` (or `_ref`) |
| MinIO / S3-compatible | `minio` | `access_key_id`, `secret_access_key`, `endpoint` |
| Cloudflare R2 | `r2` | `access_key_id`, `secret_access_key`, `account_id` |
| DigitalOcean Spaces | `spaces` | `access_key_id`, `secret_access_key` |
| Alibaba OSS | `oss` | `access_key_id`, `secret_access_key` |
| Backblaze B2 | `b2` | `access_key_id`, `secret_access_key` |
| Tencent COS | `cos` | `access_key_id`, `secret_access_key` |
| FTP | `ftp` | `username`, `password` (or `_ref`) |
| SFTP | `sftp` | `username`, `password` / `private_key` (or `_ref`) |
| SCP | `scp` | `username`, `password` / `private_key` (or `_ref`) |
| WebDAV | `webdav` | `username`, `password` (or `_ref`) |
| Local filesystem | `local` | None |

## CronJob / Job Orchestration

**Pattern:** The `BackupReconciler` manages Kubernetes `batch/v1.CronJob` resources (one per Backup CR). Each CronJob spawns `batch/v1.Job` pods on schedule.

**Scheduling:**
- Schedule source: `backup.spec.schedule.cron` (standard 5-field cron expression)
- ConcurrencyPolicy: `ForbidConcurrent` — prevents overlapping runs
- Job TTL: 60 seconds after completion
- Default job history: 3 successful, 1 failed (overridable via `spec.schedule.successfulJobsHistoryLimit` / `failedJobsHistoryLimit`)

**Update behavior:**
- On Backup manifest change (generation bump): old CronJob deleted with `OrphanPropagation` (in-progress jobs complete), new CronJob created, immediate one-off Job triggered
- Guard: `status.observedGeneration` vs `metadata.generation` prevents endless recreate loops (`internal/controller/backup_controller.go` `handleBackupUpdate()`)

**Job watching:** Controller watches `batch/v1.Job` resources and maps them back to the owning Backup via `findBackupForJob()` using CronJob owner references and name prefix matching (`<backup-name>-`).

**Pod log collection:** On job failure, operator streams last 100 lines from the `gobackup` container using the raw typed clientset (`collectPodLogs()`); stored truncated (4096 chars max) in `status.lastRun.logs`.

## Metrics & Observability

**Metrics:**
- Prometheus-format metrics exposed on `:8080` via controller-runtime's metrics server
- `ServiceMonitor` CRD supported for Prometheus Operator scraping (optional, disabled by default in Helm chart)
- Helm values: `metrics.enabled`, `serviceMonitor.enabled`

**Health probes:**
- Liveness: `GET /healthz :8081` (Ping handler)
- Readiness: `GET /readyz :8081` (Ping handler)

**Logging:**
- Structured logging via `go.uber.org/zap` through controller-runtime's `log.FromContext(ctx)` pattern
- Development mode enabled by default in operator binary

## CI/CD & Container Registry

**Operator image:**
- Registry: `ghcr.io/gobackup/gobackup-operator` (primary) and `payamqorbanpour/gobackup-operator` (Docker Hub mirror)
- Workflow: `.github/workflows/release-docker.yml` — triggers on `release-docker` branch push or GitHub release creation
- Build: Docker Buildx multi-arch (`linux/amd64`, `linux/arm64`) using `build/Dockerfile`

**Helm chart:**
- Registry: `ghcr.io/gobackup/` (OCI format)
- Workflow: `.github/workflows/release.yml` — triggers on push to `main` when `charts/**` changes
- Chart version: `0.1.3-alpha` (from `charts/gobackup-operator/Chart.yaml`)
- Installation: `helm install my-release oci://ghcr.io/gobackup/gobackup-operator --version <version>`

**Binary releases:**
- Tool: goreleaser v2 (`.goreleaser.yml`)
- Platforms: linux/amd64, linux/arm64, darwin/amd64, windows/amd64
- Checksums: `checksums.txt`

**CI tests:**
- Workflow: `.github/workflows/test.yml` — runs `go test ./...` on all PRs and branch pushes using Go 1.26.0

## Webhooks & Callbacks

**Incoming:** None defined. No admission webhooks are registered (no webhook manifests in `config/` or `api/v1/` webhook markers).

**Outgoing:** None. All communication is initiated by the operator polling the Kubernetes API.

---

*Integration audit: 2026-05-27*
