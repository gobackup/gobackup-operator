<!-- refreshed: 2026-05-27 -->
# Architecture

**Analysis Date:** 2026-05-27

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────┐
│                     User-Facing CRDs (gobackup.io/v1)               │
│                                                                      │
│  Backup (namespaced)    Database (namespaced)   Storage (namespaced) │
│  `api/v1/backup_types.go`  `api/v1/database_types.go`  `api/v1/storage_types.go` │
└────────────────────────────┬────────────────────────────────────────┘
                             │ watches + reconciles
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│               BackupReconciler (controller-runtime)                  │
│               `internal/controller/backup_controller.go`             │
│                                                                      │
│  handleBackupCreate()  handleBackupUpdate()  reconcileJobStatus()    │
└──────────┬──────────────────────────┬──────────────────────────────┘
           │                          │
           ▼                          ▼
┌────────────────────┐   ┌────────────────────────────────────────────┐
│  k8sutil.K8s       │   │  Kubernetes Native Resources                │
│  `pkg/k8sutil/`    │   │                                             │
│                    │   │  batch/v1 CronJob  (1 per Backup, same name)│
│  CreateSecret()    │   │  batch/v1 Job      (spawned by CronJob)     │
│  GetCRD()          │   │  core/v1 Secret    (gobackup.yml config)    │
│  DeleteJob()       │   │                                             │
└────────────────────┘   └────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  gobackup Job Pod  (`ghcr.io/gobackup/gobackup:v3.1.0`)             │
│  Runs: `gobackup perform`                                            │
│  Mounts Secret at /root/.gobackup/gobackup.yml                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| `BackupReconciler` | Sole controller; watches Backup CRDs and manages their lifecycle | `internal/controller/backup_controller.go` |
| `Backup` CRD | Declares what to back up (DatabaseRefs, StorageRefs) and when (Schedule.Cron) | `api/v1/backup_types.go` |
| `Database` CRD | Defines database connection config (postgresql, mysql, redis, mongodb, etc.) | `api/v1/database_types.go` |
| `Storage` CRD | Defines storage backend config (s3, gcs, azure, sftp, minio, etc.) | `api/v1/storage_types.go` |
| `k8sutil.K8s` | Low-level Kubernetes operations: Secret CRUD, CRD lookups, Job deletion | `pkg/k8sutil/` |
| `CreateSecret` | Fetches Database+Storage CRDs, resolves SecretRef fields, marshals `gobackup.yml`, writes a `core/v1 Secret` | `pkg/k8sutil/secret.go` |
| `GetCRD` | Fetches any namespaced CRD via the dynamic client | `pkg/k8sutil/crd.go` |
| `cmd/main.go` | Wires scheme, manager, clients, registers `BackupReconciler` | `cmd/main.go` |

## Pattern Overview

**Overall:** Kubebuilder v4 operator — single reconciler, level-triggered reconciliation

**Key Characteristics:**
- One controller (`BackupReconciler`) owns all three CRD kinds (Backup, Database, Storage); Database and Storage are passive read-only references, not reconciled themselves.
- CronJob-per-Backup: each `Backup` resource maps 1:1 to a `batch/v1 CronJob` with the same name and namespace.
- Configuration is materialised into a `core/v1 Secret` (named identically to the Backup) containing `gobackup.yml`, which the CronJob mounts at `/root/.gobackup`.
- Spec-change detection uses `backup.Status.ObservedGeneration` vs `backup.Generation` to avoid re-creating the CronJob on every routine reconcile.
- `controllerutil.SetControllerReference` makes the Backup the owner of the CronJob, enabling cascading deletion.

## Layers

**API Layer:**
- Purpose: Type definitions for all three CRDs; Kubebuilder marker annotations drive code generation
- Location: `api/v1/`
- Contains: `*_types.go`, `groupversion_info.go`, `zz_generated.deepcopy.go`
- Depends on: `k8s.io/apimachinery`, `k8s.io/api/core/v1`
- Used by: controller, `cmd/main.go`, `pkg/k8sutil/secret.go`

**Controller Layer:**
- Purpose: Reconcile loop driving the desired→actual state for Backup resources
- Location: `internal/controller/backup_controller.go`
- Contains: `BackupReconciler`, reconcile helpers, CronJob builder, Job status tracker
- Depends on: `api/v1`, `pkg/k8sutil`, `sigs.k8s.io/controller-runtime`, `k8s.io/client-go`
- Used by: `cmd/main.go` (registration only)

**K8s Utility Layer:**
- Purpose: Kubernetes API interactions that are too complex for inline controller code
- Location: `pkg/k8sutil/`
- Contains: `K8s` struct, `CreateSecret`, `GetCRD`, `DeleteJob`
- Depends on: `api/v1`, `k8s.io/client-go`, `k8s.io/apimachinery`, `gopkg.in/yaml.v2`
- Used by: controller layer

**Entry Point:**
- Purpose: Manager bootstrap, scheme registration, client wiring
- Location: `cmd/main.go`
- Depends on: all other layers

## Data Flow

### Backup CREATE Path

1. User applies a `Backup` manifest → API server stores it and bumps `metadata.generation=1`
2. `BackupReconciler.Reconcile` is triggered (`internal/controller/backup_controller.go:78`)
3. `r.Get` fetches the `Backup` object; checks no CronJob exists → routes to `handleBackupCreate`
4. Validates `spec.schedule.cron` (5-field check) and that `databaseRefs`/`storageRefs` are non-empty
5. `r.K8s.CreateSecret(ctx, backup)` (`pkg/k8sutil/secret.go:36`):
   - Iterates `DatabaseRefs`: calls `GetCRD` via dynamic client to fetch each `Database` CRD
   - Resolves `*_ref` SecretKeySelector fields by fetching the referenced `core/v1 Secrets`
   - Iterates `StorageRefs`: same pattern for `Storage` CRDs
   - Marshals a `gobackup.yml` YAML and writes it to a `core/v1 Secret` named after the Backup
6. `r.createCronJob(ctx, backup)` (`internal/controller/backup_controller.go:409`):
   - Builds `batchv1.CronJob` using `buildJobTemplate` — image from env `BACKUP_JOB_IMAGE` or `huacnlee/gobackup:latest`
   - Sets `ConcurrencyPolicy: ForbidConcurrent`, `TTLSecondsAfterFinished: 60`
   - Mounts the Secret as a volume at `/root/.gobackup`
   - Calls `controllerutil.SetControllerReference(backup, cronJob, r.Scheme)`
   - Creates the CronJob via `r.Create`
7. `r.recordObservedGeneration` persists `Status.ObservedGeneration = backup.Generation` via status patch
8. `r.reconcileJobStatus` runs (finds no Jobs yet, no-ops)
9. Returns `ctrl.Result{}`

### Backup UPDATE Path (spec changed)

1. User patches the `Backup` manifest → API server bumps `metadata.generation`
2. `BackupReconciler.Reconcile` triggered; CronJob exists → routes to `handleBackupUpdate`
3. Guard: `backup.Generation == backup.Status.ObservedGeneration` → skip (no-op for status-only writes)
4. `r.K8s.CreateSecret` refreshes the Secret with new config
5. `r.deleteCronJob` deletes old CronJob with `PropagationPolicy: Orphan` (preserves in-progress Jobs)
6. `r.createCronJob` creates a new CronJob from the updated spec
7. `r.hasRunningBackupJob` checks for Pending/Running Jobs prefixed `<backup-name>-`
8. If none running: `r.triggerManualBackupJob` creates a one-off `batch/v1 Job` owned by the new CronJob
9. `r.recordObservedGeneration` stamps the new generation
10. `r.reconcileJobStatus` updates `Status.LastRun`, `Status.RecentRuns`, counters

### Job Status Tracking

1. `batch/v1 Job` completion triggers `findBackupForJob` (watches Jobs, maps via CronJob owner reference)
2. Re-enqueues the owning Backup for reconciliation
3. `reconcileJobStatus` lists all Jobs with name prefix `<backup-name>-`, picks the most recent
4. Builds `BackupRunStatus`; on failure, collects last 100 log lines from the `gobackup` container via `Clientset.CoreV1().Pods(...).GetLogs`
5. Writes updated `BackupStatus` via `r.Status().Update`; keeps a sliding window of 5 runs in `Status.RecentRuns`
6. If `LastRun.Phase` is `Running` or `Pending`, requeues after 30 seconds

**State Management:**
- No in-memory state; all state lives in `Backup.Status` sub-resource (etcd-backed)
- `ObservedGeneration` is the idempotency key preventing spurious CronJob recreations

## Key Abstractions

**`BackupReconciler`:**
- Purpose: Single reconciler for the entire operator
- Examples: `internal/controller/backup_controller.go`
- Pattern: controller-runtime `Reconciler` interface; `SetupWithManager` registers `For(&Backup{})`, `Owns(&CronJob{})`, `Watches(&Job{}, ...)` 

**`k8sutil.K8s`:**
- Purpose: Wrapper holding `*kubernetes.Clientset` and `*dynamic.DynamicClient` for raw API access
- Examples: `pkg/k8sutil/k8sutil.go`, `pkg/k8sutil/secret.go`, `pkg/k8sutil/crd.go`
- Pattern: Receiver methods on a plain struct; injected into `BackupReconciler` at startup

**`BackupConfig` / `Model` (gobackup.yml schema):**
- Purpose: In-memory representation of the YAML config that gobackup reads
- Examples: `pkg/k8sutil/secret.go:17-33`
- Pattern: Go structs marshalled via `gopkg.in/yaml.v2`; field names match gobackup's expected config keys

**`StorageRef` / `DatabaseRef`:**
- Purpose: References from a Backup to named Database or Storage CRDs in the same namespace
- Examples: `api/v1/backup_types.go:68-82`
- Pattern: Resolved dynamically at reconcile time via `GetCRD` (dynamic client), not via typed informers

## Entry Points

**Operator binary:**
- Location: `cmd/main.go`
- Triggers: `controller-runtime` Manager; signal handler (`ctrl.SetupSignalHandler`)
- Responsibilities: Register scheme (`gobackup.io/v1` + core k8s), create `K8s` clients, instantiate `BackupReconciler`, add health/readiness probes, start manager

**`BackupReconciler.Reconcile`:**
- Location: `internal/controller/backup_controller.go:78`
- Triggers: Any change to a watched `Backup`, owned `CronJob`, or watched `Job`
- Responsibilities: Route to create or update handler; always run `reconcileJobStatus` after; conditionally requeue if a job is in-flight

## Architectural Constraints

- **Threading:** Single-threaded event loop per resource type (controller-runtime default); no explicit goroutines in controller code
- **Global state:** `scheme` and `setupLog` package-level vars in `cmd/main.go`; all other state flows through reconciler structs
- **Circular imports:** None detected; `api/v1` → no internal deps; `pkg/k8sutil` → `api/v1`; `internal/controller` → both
- **CRD version:** All resources are `gobackup.io/v1`; domain is `gobackup.io`
- **Namespace scope:** All CRDs are namespaced; the operator cluster-role watches across all namespaces
- **Dynamic client dependency:** `Database` and `Storage` CRDs are fetched via the dynamic client at reconcile time, not cached via typed informers — changes to those CRDs do not automatically re-trigger Backup reconciliation; only a change to the Backup spec or its owned CronJob/Jobs does

## Anti-Patterns

### Direct Kubernetes Secret writes bypassing controller-runtime cache

**What happens:** `pkg/k8sutil/secret.go` uses `k.Clientset.CoreV1().Secrets(namespace).Create/Update` (raw typed client) instead of `r.Client.Create/Update` (controller-runtime cached client).
**Why it's wrong:** The cached client and the raw clientset can diverge transiently; the Secret is not added to the controller's informer cache, so owner-reference-based watches may miss it briefly.
**Do this instead:** Prefer `r.Create(ctx, secret)` / `r.Update(ctx, secret)` via the controller-runtime `client.Client` for resources the controller needs to track.

### `regcred` image pull secret is hardcoded

**What happens:** `buildJobTemplate` always appends `ImagePullSecrets: [{Name: "regcred"}]` (`internal/controller/backup_controller.go:358`).
**Why it's wrong:** Any cluster that does not have a secret named `regcred` in the Backup's namespace will get pod scheduling failures silently.
**Do this instead:** Make `imagePullSecrets` configurable via the `Backup` spec or a Helm value (the `backupJob.image` pattern in `charts/gobackup-operator/values.yaml` already exists but `imagePullSecrets` is not exposed).

## Error Handling

**Strategy:** Errors bubble up from helpers via `fmt.Errorf("...: %w", err)` wrapping. The reconciler returns `(ctrl.Result{}, err)` for fatal errors, triggering exponential back-off requeue by the manager. Conflict errors on status updates are swallowed and retried on the next natural reconcile.

**Patterns:**
- `client.IgnoreNotFound(err)` used on Backup fetch — graceful handling of already-deleted resources
- `errors.IsAlreadyExists(err)` during CronJob recreation → explicit `RequeueAfter: 2s` while old CronJob terminates
- `errors.IsConflict(err)` on status update → logged, returns nil (next reconcile will retry)
- Pod log collection failures during status reporting are non-fatal; error message is written to `Status.LastRun.Logs`

## Cross-Cutting Concerns

**Logging:** `sigs.k8s.io/controller-runtime/pkg/log` (`zap` backend); structured key-value pairs; `logger.V(1)` for verbose/debug messages
**Validation:** In-controller validation (`validateCronExpression`, `validateBackupSpec`) before acting; CRD-level CEL validation on `DatabaseSpec` (type-specific field guards in `database_types.go:25-40`)
**Authentication:** ServiceAccount + ClusterRole/ClusterRoleBinding; RBAC markers on reconciler generate `config/rbac/role.yaml`
**Metrics:** Prometheus endpoint on `:8080`; ServiceMonitor optional (disabled by default in Helm values)
**Health probes:** `/healthz` (liveness) and `/readyz` (readiness) on `:8081`
**Leader election:** Disabled by default in manager config; enabled via `--leader-elect` flag; Helm value `leaderElection.enabled: true`

---

*Architecture analysis: 2026-05-27*
