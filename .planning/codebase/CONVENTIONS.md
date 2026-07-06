# Coding Conventions

**Analysis Date:** 2026-05-27

## Naming Patterns

**Packages:**
- `controller` — single package for all reconcilers in `internal/controller/`
- `k8sutil` — utility helpers in `pkg/k8sutil/`
- `v1` — CRD type definitions in `api/v1/`
- Package names match directory names; no plurals

**Types:**
- CRD resource types use `PascalCase`: `Backup`, `Database`, `Storage`
- Spec types follow `<Resource>Spec`: `BackupSpec`, `DatabaseSpec`, `StorageSpec`
- Status types follow `<Resource>Status`: `BackupStatus`, `DatabaseStatus`
- Sub-status structs follow `<Resource><Sub>Status`: `BackupRunStatus`
- Schedule nested types: `BackupSchedule`
- List types follow `<Resource>List`: `BackupList`, `DatabaseList`, `StorageList`
- Reconciler types follow `<Resource>Reconciler`: `BackupReconciler`

**Functions (controller methods):**
- Entry point: `Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)` (standard controller-runtime signature)
- Sub-handlers: `handle<Resource><Operation>` — e.g., `handleBackupCreate`, `handleBackupUpdate`
- Creators: `create<Resource>` — e.g., `createCronJob`
- Deleters: `delete<Resource>` — e.g., `deleteCronJob`
- Builders: `build<Resource>` — e.g., `buildJobTemplate`, `buildRunStatus`
- Validators: `validate<Subject>` — e.g., `validateBackupSpec`, `validateCronExpression`
- Helpers: descriptive verbs — `addToRecentRuns`, `truncateString`, `ensureOwnerReference`, `recordObservedGeneration`
- Setup: `SetupWithManager(mgr ctrl.Manager) error` (standard controller-runtime)

**Variables:**
- Local loggers: `logger := log.FromContext(ctx)` (obtained at the top of each method)
- Kubernetes objects: type matches kind, lowercase: `backup`, `cronJob`, `job`, `secret`
- Counters and limits: SCREAMING_SNAKE_CASE for package-level constants: `MaxRecentRuns`, `MaxLogSize`, `MaxMessageSize`

**JSON tags:**
- CRD spec fields use `camelCase` JSON tags: `databaseRefs`, `storageRefs`, `compressWith`
- Exception: gobackup config fields that must match gobackup YAML format use `snake_case`: `password_ref`, `access_key_id`, `before_script`
- Pointer types used for optional fields: `*string`, `*int`, `*bool`, `*int32`, `*int64`

## Code Style

**Formatting:**
- Standard `go fmt ./...` via `make fmt` before every build and test
- Enforced in the `build`, `test`, `test-unit`, and `test-integration` Makefile targets

**Linting:**
- `golangci-lint` v1.54.2 via `make lint` / `make lint-fix`
- No `.golangci.yml` config file — uses golangci-lint defaults
- `go vet ./...` runs as part of `make vet`, included in build/test chains

**Vet:**
- `go vet ./...` runs before every `test`, `build`, and `run` target

## Import Organization

**Three groups, separated by blank lines:**

1. Standard library
2. Third-party and Kubernetes libraries (k8s.io, sigs.k8s.io, gopkg.in)
3. Internal project packages (github.com/gobackup/gobackup-operator/...)

**Example from `internal/controller/backup_controller.go`:**
```go
import (
    "bytes"
    "context"
    "fmt"
    "io"
    "os"
    "strings"
    "time"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/kubernetes"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/handler"
    "sigs.k8s.io/controller-runtime/pkg/log"

    backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
    "github.com/gobackup/gobackup-operator/pkg/k8sutil"
)
```

**Import aliases:**
- `ctrl` for `sigs.k8s.io/controller-runtime`
- `batchv1`, `corev1`, `metav1` for versioned Kubernetes API packages
- `backupv1` for internal CRD types
- `logf` for `sigs.k8s.io/controller-runtime/pkg/log` in test files

## Kubebuilder Marker Usage

**Package-level markers (in `api/v1/groupversion_info.go`):**
```go
// +kubebuilder:object:generate=true
// +groupName=gobackup.io
package v1
```

**Resource type markers (directly above the primary struct, no space before `//`):**
```go
//+kubebuilder:resource:shortName=backup
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
type Backup struct { ... }

//+kubebuilder:object:root=true
type BackupList struct { ... }
```

**Field-level validation markers (inline above the field, with space after `//`):**
```go
// +kubebuilder:validation:Enum=postgresql;mysql;mariadb;mongodb;redis;mssql;influxdb;etcd
Type string `json:"type"`

// +kubebuilder:validation:MaxItems=5
RecentRuns []BackupRunStatus `json:"recentRuns,omitempty"`

// +kubebuilder:validation:Enum=copy;sync
Mode *string `json:"mode,omitempty"`
```

**Struct-level CEL validation markers (above the spec struct):**
```go
// +kubebuilder:validation:XValidation:rule="self.type == 'redis' || !has(self.config.mode)",message="..."
type DatabaseSpec struct { ... }
```

**RBAC markers (above Reconcile, with space after `//`):**
```go
// +kubebuilder:rbac:groups=gobackup.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.io,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
```

**Scaffold markers (in `cmd/main.go` and `suite_test.go`, no space after `//`):**
```go
//+kubebuilder:scaffold:imports
//+kubebuilder:scaffold:scheme
//+kubebuilder:scaffold:builder
```

**Convention:** resource-type markers use `//+` (no space); field/RBAC/validation markers use `// +` (with space).

## Error Handling

**Pattern 1 — not found on fetch (idiomatic controller-runtime):**
```go
if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
}
```

**Pattern 2 — not found tolerated (delete-idempotent):**
```go
if err := r.Delete(ctx, cronJob, ...); err != nil {
    if errors.IsNotFound(err) {
        return nil
    }
    return fmt.Errorf("failed to delete CronJob %s/%s: %w", ...)
}
```

**Pattern 3 — conflict on status update (treated as non-fatal):**
```go
if err := r.Status().Update(ctx, backup); err != nil {
    if errors.IsConflict(err) {
        logger.Info("Conflict updating backup status, will retry on next reconciliation")
        return nil
    }
    return fmt.Errorf("failed to update backup status: %w", err)
}
```

**Pattern 4 — AlreadyExists triggers explicit requeue:**
```go
if errors.IsAlreadyExists(err) {
    logger.V(1).Info("Old CronJob still terminating, requeueing")
    return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}
```

**Error wrapping:** All errors returned from sub-functions use `fmt.Errorf("description: %w", err)` — sentinel wrapping throughout `pkg/k8sutil/`.

**Fatal startup errors:** `os.Exit(1)` in `cmd/main.go` after logging with `setupLog.Error`.

## Requeue Patterns

| Situation | Return |
|-----------|--------|
| Not found (deleted) | `ctrl.Result{}, client.IgnoreNotFound(err)` |
| Deletion in progress | `ctrl.Result{}, nil` |
| Unrecoverable error | `ctrl.Result{}, err` (controller-runtime retries with backoff) |
| Job still terminating | `ctrl.Result{RequeueAfter: 2 * time.Second}, nil` |
| Job Running or Pending | `result.RequeueAfter = 30 * time.Second` (set before final return) |
| Success, no requeue needed | `ctrl.Result{}, nil` |

## Logging

**Framework:** `logr` via `sigs.k8s.io/controller-runtime/pkg/log`, backed by zap.

**Logger acquisition:** Always `logger := log.FromContext(ctx)` at the top of the function body — never stored on the struct.

**Verbosity levels:**
- `logger.Info(...)` — normal reconcile events (entry, state transitions, success)
- `logger.Error(err, "message", ...)` — errors that prevent reconcile from succeeding
- `logger.V(1).Info(...)` — verbose debug info (no-op changes, skipped operations)

**Structured key-value pairs:** Always use key-value pairs as trailing arguments:
```go
logger.Info("Reconciling Backup", "namespace", req.Namespace, "name", req.Name)
logger.Error(err, "Failed to create CronJob during Backup create")
logger.V(1).Info("Backup spec unchanged, keeping existing CronJob", "generation", backup.Generation)
```

**Setup logger:** `setupLog = ctrl.Log.WithName("setup")` — module-level variable used only in `cmd/main.go` startup path.

## Function Design

**Size:** Methods are split into focused sub-handlers. `Reconcile` itself is ~60 lines; complex logic is delegated to `handleBackupCreate`, `handleBackupUpdate`, `reconcileJobStatus`.

**Context:** `ctx context.Context` is always the first parameter on any function that calls Kubernetes API.

**Receiver:** All controller methods use pointer receiver `(r *BackupReconciler)`. All `k8sutil` helpers use `(k *K8s)`.

**Return values:** Functions return `(ctrl.Result, error)` from handlers, plain `error` from helpers.

## Comments

**Function-level doc comments:** Every exported function and every private reconciler sub-method has a doc comment starting with the function name:
```go
// handleBackupCreate handles the creation of a new Backup resource.
// This method is called when a Backup CRD is first created.
func (r *BackupReconciler) handleBackupCreate(...) ...
```

**Inline comments:** Used to explain non-obvious control flow and reconciler logic (generation comparison, orphan propagation).

**CRD field comments:** Every CRD field in `api/v1/` has a doc comment explaining its purpose and which database/storage types it applies to.

**Scaffold markers:** `//+kubebuilder:scaffold:*` comments mark injection points for kubebuilder code generation — do not remove.

## Module Design

**Exports:**
- `api/v1` exports all CRD types and `AddToScheme`
- `pkg/k8sutil` exports `K8s` struct and `NewClient`, `NewDynamicClient` constructors
- `internal/controller` exports `BackupReconciler` and `SetupWithManager`

**Barrel files:** Not used. Each file exports what it defines; no re-export files.

**`init()` pattern:** Each `api/v1/*_types.go` file registers its types with the scheme builder via `init()`:
```go
func init() {
    SchemeBuilder.Register(&Backup{}, &BackupList{})
}
```

---

*Convention analysis: 2026-05-27*
