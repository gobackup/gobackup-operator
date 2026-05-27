# Technology Stack

**Analysis Date:** 2026-05-27

## Languages

**Primary:**
- Go 1.26.0 - All operator logic, CRD types, and controller implementation

**Secondary:**
- YAML - CRD manifests, Helm chart templates, Kustomize overlays, gobackup config generation

## Runtime

**Environment:**
- Linux container (BusyBox minimal base image in production)
- In-cluster Kubernetes pod via `/manager` binary entrypoint

**Package Manager:**
- Go modules (`go.mod` / `go.sum`)
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- `sigs.k8s.io/controller-runtime` v0.23.1 - Operator reconcile loop, manager, scheme, envtest
- Kubebuilder v4 scaffold (`go.kubebuilder.io/v4`) - Project layout and code generation conventions (`PROJECT` file)

**Kubernetes Client:**
- `k8s.io/client-go` v0.35.1 - Raw Kubernetes typed clientset and dynamic client
- `k8s.io/api` v0.35.1 - Core API types (Pod, Secret, Job, CronJob)
- `k8s.io/apimachinery` v0.35.1 - API machinery primitives (runtime, meta, unstructured)
- `k8s.io/apiextensions-apiserver` v0.35.1 (indirect) - CRD validation support

**Testing:**
- `github.com/onsi/ginkgo/v2` v2.27.2 - BDD test runner (`internal/controller/suite_test.go`)
- `github.com/onsi/gomega` v1.38.2 - Matcher assertions
- `sigs.k8s.io/controller-runtime/tools/setup-envtest` - Downloads envtest binaries for k8s 1.28.0

**Build/Dev:**
- Kustomize v5.2.1 - Kubernetes manifest overlay toolchain (`bin/kustomize`)
- controller-gen (latest) - Generates CRDs and DeepCopy methods from Go markers (`bin/controller-gen`)
- goreleaser v2 - Multi-platform binary release and archive creation (`.goreleaser.yml`)
- golangci-lint v1.54.2 - Static analysis (`bin/golangci-lint`)
- Docker / Buildx - Container image build and multi-arch push
- kind - Local cluster for development and integration testing

## Key Dependencies

**Critical:**
- `sigs.k8s.io/controller-runtime` v0.23.1 - The entire reconciliation framework; controls how the operator watches and reacts to Kubernetes resources
- `k8s.io/client-go` v0.35.1 - Raw typed clientset used in `pkg/k8sutil` for Secret and Pod log API calls not exposed via controller-runtime's generic client
- `gopkg.in/yaml.v2` v2.4.0 - Serializes `BackupConfig` structs into `gobackup.yml` format stored in Kubernetes Secrets

**Infrastructure:**
- `github.com/prometheus/client_golang` v1.23.2 - Metrics endpoint exposed on `:8080` via controller-runtime metrics server
- `go.uber.org/zap` v1.27.1 - Structured logging via controller-runtime's zap integration
- `github.com/Masterminds/semver/v3` v3.4.0 (indirect) - Semver parsing used internally

## Configuration

**Environment Variables:**
- `BACKUP_JOB_IMAGE` - Docker image used for per-backup CronJob/Job pods. Defaults to `huacnlee/gobackup:latest` when unset. Set to `ghcr.io/gobackup/gobackup:v3.1.0` in the Helm chart (`charts/gobackup-operator/values.yaml`)
- Standard `KUBECONFIG` / in-cluster service account token - Kubernetes API access (`pkg/k8sutil/k8sutil.go`)

**Runtime Flags (cmd/main.go):**
- `--metrics-bind-address` (default `:8080`) - Prometheus metrics endpoint
- `--health-probe-bind-address` (default `:8081`) - Liveness/readiness endpoints
- `--leader-elect` (default `false`) - Leader election; set to `true` by Helm chart

**Build:**
- `build/Dockerfile` - Multi-stage: Go 1.26.0 builder + BusyBox minimal runtime
- `.goreleaser.yml` - Produces binaries for linux/amd64, linux/arm64, darwin/amd64, windows/amd64
- `Makefile` - Primary developer interface for build, test, generate, lint, and kind cluster workflows

## Platform Requirements

**Development:**
- Go 1.26.0
- Docker (or compatible container tool)
- kind (for local cluster testing)
- kubectl
- `make manifests generate` must run after any change to `api/v1/*.go` types

**Production:**
- Kubernetes >= 1.19.0 (per `charts/gobackup-operator/Chart.yaml`)
- CRDs installed: `gobackup.io_backups`, `gobackup.io_databases`, `gobackup.io_storages`
- Deployment target: container registry `ghcr.io/gobackup/gobackup-operator` (operator binary) and `ghcr.io/gobackup/gobackup` (backup job image)

---

*Stack analysis: 2026-05-27*
