# Codebase Concerns

**Analysis Date:** 2026-05-27

---

## Tech Debt

**Helm chart CRDs are not auto-synced from `config/crd/bases/`:**
- Issue: `make manifests` writes generated CRDs only to `config/crd/bases/`. The Helm chart copies at `charts/gobackup-operator/crds/` must be hand-synced. The databases and storages CRDs currently differ by 130–265 lines (formatting, added validation rules, `SecretKeySelector` field defaults). The backups CRD happens to be identical, but that is coincidental.
- Files: `charts/gobackup-operator/crds/gobackup.io_databases.yaml`, `charts/gobackup-operator/crds/gobackup.io_storages.yaml`, `config/crd/bases/` (all three)
- Impact: Helm users who upgrade the chart may run with stale CRD schemas — missing new fields, missing CEL validation rules, or triggering Helm "schema drift prunes new fields" behavior on re-apply.
- Fix approach: Add a `make sync-helm-crds` target that copies `config/crd/bases/*.yaml` into `charts/gobackup-operator/crds/` and wire it into `make manifests`. Gate CI on a diff check between the two directories.

**Hardcoded `regcred` image pull secret:**
- Issue: `buildJobTemplate` unconditionally appends `ImagePullSecrets: [{Name: "regcred"}]` to every backup Job pod spec.
- Files: `internal/controller/backup_controller.go:360`
- Impact: All clusters that do not have a secret named `regcred` in each namespace where Backups run will encounter `ImagePullBackOff` on every scheduled job. There is no configuration surface to change this value.
- Fix approach: Make the image pull secret name(s) configurable via an operator flag or a field on the `BackupSpec`. Default to empty (no pull secret) or read from the Helm `values.yaml`.

**`Development: true` logger baked into production binary:**
- Issue: `cmd/main.go:66` passes `Development: true` to the zap logger. This enables panic on `DPanic`, colorized output, and callerSkip adjustments that are inappropriate for production pods.
- Files: `cmd/main.go:65-66`
- Impact: Log output is harder to parse by log aggregators (non-JSON). `DPanic` calls will crash the operator in production.
- Fix approach: Wire development mode to an `--zap-devel` flag (the standard controller-runtime convention); default it to `false`.

**Stale RBAC resources (`postgresqls`, `s3s`):**
- Issue: Both `config/rbac/role.yaml:76-77` and `charts/gobackup-operator/templates/clusterrole.yaml:81-82` grant `get/list/watch` on `postgresqls` and `s3s` resource groups, which no longer exist as separate CRDs. The operator now uses the unified `databases` and `storages` resources.
- Files: `config/rbac/role.yaml`, `charts/gobackup-operator/templates/clusterrole.yaml`
- Impact: Dead RBAC entries expand the operator's effective permission surface without benefit.
- Fix approach: Remove the `postgresqls` and `s3s` entries from both files and from the `+kubebuilder:rbac` markers at `internal/controller/backup_controller.go:67-68`.

**Incomplete camelCase-to-snake_case field mapping:**
- Issue: `pkg/k8sutil/secret.go:89-108` performs only a handful of explicit key renames when building the gobackup YAML config. Any CRD field that arrives in camelCase from the Kubernetes unstructured object and is not listed in the `switch` block is passed through verbatim — and gobackup will not recognize it.
- Files: `pkg/k8sutil/secret.go:89-108`, `pkg/k8sutil/secret.go:147-165`
- Impact: Silent misconfiguration: a new `DatabaseConfig` or `StorageConfig` field added to the CRD types without a corresponding `switch` case will be silently dropped from the generated `gobackup.yml`. Backups succeed but the option has no effect.
- Fix approach: Replace the ad-hoc `switch` with a systematic camelCase→snake_case converter (e.g. using `github.com/iancoleman/strcase`) or enforce snake_case JSON tags throughout the types.

**Misleading comment on TTL constant:**
- Issue: `internal/controller/backup_controller.go:340-342` has the comment `// Hardcoded to 1 second (1 second)` but the actual value is `int32(60)` seconds.
- Files: `internal/controller/backup_controller.go:340-342`
- Impact: Minor developer confusion; not a runtime defect.
- Fix approach: Update the comment to match the value.

**Duplicate import alias in `cmd/main.go`:**
- Issue: `cmd/main.go:36-37` imports the same package `github.com/gobackup/gobackup-operator/api/v1` under two aliases (`backupv1` and `gobackupiov1`), and both are passed to `AddToScheme` at `cmd/main.go:51-52`. This registers the same types twice (idempotent but wasteful and confusing).
- Files: `cmd/main.go:36-52`
- Impact: Confusion for developers; `gobackupiov1` is never used after scheme registration.
- Fix approach: Remove the second import alias.

---

## Known Issues / Recent Fixes (Fragile History)

**CronJob delete-recreate race: "old CronJob still terminating" on rapid updates:**
- Symptoms: When a Backup manifest is edited quickly twice in succession, the second reconcile tries to create the new CronJob before the old one has fully terminated. The controller returns `{RequeueAfter: 2s}` and retries, but during that window status may be stale.
- Files: `internal/controller/backup_controller.go:246-252`
- Trigger: Two consecutive `kubectl apply` or `kubectl edit` calls within 2 seconds on the same Backup resource.
- History: This scenario drove PR #77 (commit `77bfb2c`). The requeue-on-`AlreadyExists` is the current mitigation.
- Workaround: Wait for the requeue cycle; the operator recovers on its own.

**`reconcileJobStatus` swallows status-update conflicts silently:**
- Issue: At `internal/controller/backup_controller.go:638-641`, a `StatusConflict` error from `Status().Update` is silently discarded with `return nil`. This means a status update may be lost during periods of high reconcile activity (e.g., right after a manifest change triggers both `recordObservedGeneration` and `reconcileJobStatus`).
- Files: `internal/controller/backup_controller.go:638-641`
- Impact: `Backup.Status.Phase`, `SuccessCount`, `FailureCount` may be transiently stale or permanently wrong if two status writes race.
- Fix approach: Use `Status().Patch` with `MergeFrom` (as done in `recordObservedGeneration`) instead of `Status().Update`; patches are idempotent under conflict.

**Two concurrent status writes per reconcile (patch + update):**
- Issue: Every reconcile that goes through create/update path calls `recordObservedGeneration` (which does a `Status().Patch`) and then `reconcileJobStatus` (which does a `Status().Update`) on the same object within the same reconcile loop tick.
- Files: `internal/controller/backup_controller.go:183-186`, `internal/controller/backup_controller.go:272-277`, `internal/controller/backup_controller.go:637`
- Impact: The second write races against the first. The comment at line 526 acknowledges this by calling the patch "a merge patch so it does not conflict… within the same reconcile" — but `Status().Update` at line 637 does a full replace, making that guarantee moot.
- Fix approach: Consolidate all status mutations into a single `Status().Patch` call at the end of reconcile, or refetch the object before the final update.

---

## Security Considerations

**Plaintext credentials stored in CRD spec (visible in etcd and RBAC-authorized reads):**
- Risk: `DatabaseConfig.Password` (`api/v1/database_types.go:69`), `DatabaseConfig.Token` (`api/v1/database_types.go:114`), `StorageConfig.Password`, `StorageConfig.AccessKeyID`, `StorageConfig.SecretAccessKey`, `StorageConfig.Credentials` (GCS JSON), `StorageConfig.ClientSecret` (Azure) are all plain `*string` fields with no `kubebuilder:validation` `x-kubernetes-preserve-unknown-fields` secret annotation or encrypt-at-rest enforcement.
- Files: `api/v1/database_types.go:69,114`, `api/v1/storage_types.go:58,96,104,130,160`
- Current mitigation: `*Ref` variants using `corev1.SecretKeySelector` are provided as alternatives; documentation encourages their use.
- Recommendations: Mark the plaintext fields `// +kubebuilder:validation:Optional` with a deprecation notice and add a CEL validation rule that rejects specs where both the plaintext field and the `_ref` field are set simultaneously. Consider a future version that removes the plaintext fields entirely.

**ClusterRole grants cluster-wide secret write access:**
- Risk: The operator's ClusterRole (`config/rbac/role.yaml:22-31`) grants `create/delete/get/list/patch/update/watch` on `secrets` across the entire cluster with no namespace restriction. Because the ClusterRoleBinding attaches this to the operator's ServiceAccount, any namespace escalation or pod-escape vulnerability in the operator binary gives full cluster secret access.
- Files: `config/rbac/role.yaml:22-31`, `config/rbac/role_binding.yaml`
- Current mitigation: None. The Helm chart replicates this as a ClusterRole at `charts/gobackup-operator/templates/clusterrole.yaml:11-19`.
- Recommendations: If the operator is intended to be namespace-scoped (all Backup, Database, and Storage resources are namespaced), replace the ClusterRole+ClusterRoleBinding with a Role+RoleBinding per namespace, or scope secret permissions to the operator's own namespace.

**Operator RBAC grants `create` on Backup resources:**
- Risk: `config/rbac/role.yaml:47-57` includes `create` and `delete` verbs on `backups`. The controller never creates Backup CRs itself; these verbs are unnecessary and represent an over-permissioned surface.
- Files: `config/rbac/role.yaml:47-57`
- Recommendations: Remove `create` and `delete` verbs from the `backups` rule; keep only `get/list/watch/update/patch`.

**gobackup credentials stored verbatim in a Kubernetes Secret (gobackup.yml):**
- Risk: `pkg/k8sutil/secret.go` marshals all resolved credentials (including plaintext values fetched from referenced Secrets) into a single `gobackup.yml` Secret. Any pod or ServiceAccount that can `get secrets` in the Backup's namespace can read the full credential bundle.
- Files: `pkg/k8sutil/secret.go:220-260`
- Current mitigation: The generated Secret has an owner reference to the Backup object, so it is garbage-collected on Backup deletion.
- Recommendations: Add a `kubernetes.io/enforce-mountable-secrets` annotation to the generated Secret and ensure it is not viewable by resources other than the Job pods.

---

## Performance Bottlenecks

**`reconcileJobStatus` lists all Jobs in the namespace on every reconcile:**
- Problem: `internal/controller/backup_controller.go:545-548` calls `r.List` for all `batchv1.Job` in the namespace with no label selector. Every reconcile — including the frequent 30-second requeueing when a backup is running — incurs a full namespace Job list.
- Files: `internal/controller/backup_controller.go:545-548`
- Cause: Jobs created by CronJob do not carry a label identifying the parent Backup; the controller falls back to name-prefix matching.
- Improvement path: Add a label (e.g. `gobackup.io/backup-name: <name>`) to all generated Jobs (in both `createCronJob` job template and `triggerManualBackupJob`), then filter with `client.MatchingLabels` in the List call.

**`hasRunningBackupJob` issues a second full Job list per update reconcile:**
- Problem: `internal/controller/backup_controller.go:476-492` performs another unbounded `r.List` immediately before the Job creation step during update reconciles.
- Files: `internal/controller/backup_controller.go:476-492`
- Improvement path: Cache the Job list from `reconcileJobStatus` and pass it as a parameter, or consolidate both into a single list call early in reconcile.

**No watch predicate to filter generation-unchanged events:**
- Problem: The controller has no `WithEventFilter` / `predicate.GenerationChangedPredicate` on the `For(&backupv1.Backup{})` watch. Every status-only write to a Backup object (including the controller's own `recordObservedGeneration` and `reconcileJobStatus` writes) triggers a new reconcile event. This is partially mitigated by the `backup.Generation == backup.Status.ObservedGeneration` guard at line 222, but the reconcile still runs, lists Jobs, and checks status.
- Files: `internal/controller/backup_controller.go:367-374`
- Improvement path: Add `predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})` to the `For` builder to suppress status-write re-entries.

---

## Fragile Areas

**Job-to-Backup association via name prefix (`strings.HasPrefix`):**
- Files: `internal/controller/backup_controller.go:483`, `internal/controller/backup_controller.go:556`
- Why fragile: A Backup named `app` will match Jobs belonging to a Backup named `app-secondary` (prefix `app-` is a substring of `app-secondary-`). Collision is unlikely but becomes probable in namespaces with many Backups sharing a common name prefix.
- Safe modification: Add a Kubernetes label `gobackup.io/backup-name: <exact-name>` to all managed Jobs and use `client.MatchingLabels` for association instead of prefix matching.
- Test coverage: No unit tests cover the prefix-matching logic.

**Delete-then-create CronJob pattern is non-atomic:**
- Files: `internal/controller/backup_controller.go:238-253`
- Why fragile: Between `deleteCronJob` and the successful `createCronJob`, there is a window where no CronJob exists for the Backup. If the operator restarts mid-reconcile, the Backup object has `ObservedGeneration` not yet updated (it is updated _after_ creation at line 274), so the next reconcile will route as an update and attempt the delete-then-create cycle again. The `IsAlreadyExists` guard at line 246 handles the "new CronJob already exists" edge case, but not the "CronJob was never created" case after a crash between delete and create.
- Safe modification: Record `ObservedGeneration` immediately after deleting the old CronJob (or before), not after creating the new one.

**No finalizers on Backup resources:**
- Files: `internal/controller/backup_controller.go:62-73` (RBAC marker for finalizers exists but no finalizer logic is implemented), `internal/controller/backup_controller.go:89-92`
- Why fragile: When a Backup is deleted, garbage collection via owner references removes the CronJob and the generated Secret automatically. However, any orphaned Jobs (created with `PropagationPolicy(Orphan)`) are not tracked or cleaned up. They will continue running to completion but are invisible to the deleted Backup.
- Safe modification: Add a finalizer that waits for all in-flight Jobs (those prefixed with the Backup name) to reach terminal state before allowing the Backup object to be fully deleted.

**`Backup.Status.Conditions` slice is never populated:**
- Files: `api/v1/backup_types.go:129-130`, `internal/controller/backup_controller.go` (no calls to set Conditions)
- Why fragile: The `Conditions` field is declared in the CRD schema and included in generated YAML (with `+optional`), but no reconcile path ever writes to it. Tools and users that rely on standard Kubernetes Condition patterns (e.g. `kubectl wait --for=condition=Ready`) will not function.
- Safe modification: Populate at least a `Ready` or `Scheduled` Condition in `reconcileJobStatus` using `apimeta.SetStatusCondition`.

---

## Upgrade/Migration Concerns

**Single CRD version (`v1`) with no conversion webhook:**
- All three CRDs (Backup, Database, Storage) are registered at `v1` with no stored-but-not-served older version and no conversion webhook.
- Files: `api/v1/groupversion_info.go:28`, all CRD bases under `config/crd/bases/`
- Impact: Any future breaking change to CRD fields (rename, type change, removal) will require a new API version (`v2`) and a conversion webhook. The scaffolding for this does not exist today. Rolling out such a change without downtime requires careful conversion planning.
- Recommendation: Document the "v1 is stable" commitment or proactively scaffold `v1` as the hub for future conversion; add an `// +kubebuilder:storageversion` marker explicitly to all types.

**`DatabaseStatus` and `StorageStatus` are empty stubs:**
- Files: `api/v1/database_types.go:152-156`, `api/v1/storage_types.go:182-185`
- Impact: If a status subresource field is added to these types in the future, existing installations will not have that field populated and may behave inconsistently until their objects are reconciled.
- Recommendation: Add at minimum a `ObservedGeneration int64` field to both status types now, before the types reach wider use.

**`go 1.26.0` in `go.mod` is a future/unreleased Go version:**
- Files: `go.mod:3`
- Impact: Go 1.26 does not exist as of the analysis date. This line may have been set speculatively. The CI pipeline pins `go-version: '1.26.0'` as well. Most Go toolchains will accept a higher `go` directive by falling forward, but it signals the `go.mod` was not validated against a real released toolchain.
- Recommendation: Set `go 1.23` (or the actual latest stable release used in CI) and update when upgrading.

---

## Dependencies at Risk

**`gopkg.in/yaml.v2` instead of `v3`:**
- Risk: `gopkg.in/yaml.v2` is the dependency used to marshal `gobackup.yml` (`pkg/k8sutil/secret.go:8`). yaml.v2 has been in maintenance mode since 2022 and does not support YAML 1.2 features (which v3 does). The module is still actively patched for CVEs but is not receiving new features.
- Impact: If gobackup's config format requires YAML 1.2 constructs, marshaling may produce subtly wrong output. String values containing special characters may be quoted differently between v2 and v3.
- Migration plan: Migrate to `gopkg.in/yaml.v3`; the API is largely compatible with minor changes to struct tag behavior.

**`controller-runtime v0.23.1` — check for API deprecations:**
- `go.mod:8` pins `sigs.k8s.io/controller-runtime v0.23.1`. The `handler.EnqueueRequestsFromMapFunc` usage at `internal/controller/backup_controller.go:372` requires a non-nil `context.Context` parameter in its map function signature (introduced in recent controller-runtime). Confirm the signature matches v0.23.1's API to avoid a silent build break on downgrade.
- Files: `go.mod:8`, `internal/controller/backup_controller.go:376-405`

---

## Test Coverage Gaps

**No controller reconcile unit tests:**
- What's not tested: The entire `BackupReconciler.Reconcile`, `handleBackupCreate`, `handleBackupUpdate`, `reconcileJobStatus`, and `recordObservedGeneration` code paths have no test coverage. The only test file in `internal/controller/` is `suite_test.go`, which sets up the envtest environment but contains no test cases (0 `It` / `Describe` / `func Test` blocks beyond the suite runner).
- Files: `internal/controller/suite_test.go` (entire file), `internal/controller/backup_controller.go`
- Risk: Regressions in reconcile logic (e.g., the delete-recreate race, status counter increments, generation tracking) go undetected until production.
- Priority: High

**No tests for `pkg/k8sutil/secret.go`:**
- What's not tested: `CreateSecret`, `resolveSecretReferences`, `ensureOwnerReference`, `DeleteSecret` — the core credential assembly pipeline.
- Files: `pkg/k8sutil/secret.go`, `pkg/k8sutil/crd.go`
- Risk: Silent misconfiguration (wrong YAML output, missed field conversions, broken secret reference resolution) shipped without detection.
- Priority: High

**No tests for name-prefix Job association:**
- What's not tested: The `strings.HasPrefix(job.Name, backup.Name+"-")` logic at lines 483 and 556 — particularly the collision scenario described under Fragile Areas.
- Files: `internal/controller/backup_controller.go:483`, `internal/controller/backup_controller.go:556`
- Priority: Medium

**envtest binary pinned to Kubernetes 1.28.3:**
- What's not tested: `internal/controller/suite_test.go:65-67` pins envtest assets to `1.28.3`. The operator targets `k8s.io/api v0.35.1` (Kubernetes 1.35). Behaviors specific to newer API server versions (CRD validation, admission, status subresource handling) are not exercised by envtest.
- Files: `internal/controller/suite_test.go:65-67`, `Makefile` (`ENVTEST_K8S_VERSION`)
- Priority: Medium

---

*Concerns audit: 2026-05-27*
