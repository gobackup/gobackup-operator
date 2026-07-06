# Contributing to gobackup-operator

Thanks for your interest in contributing! This project is a Kubernetes
operator for [gobackup](https://github.com/gobackup/gobackup), built with
Kubebuilder v4 and controller-runtime.

> **Status:** alpha. The API (`gobackup.io/v1alpha1`) may change in breaking
> ways between releases.

## Prerequisites

- Go 1.26+
- Docker
- `kind` and `kubectl` (for local clusters and e2e tests)
- Project tooling is installed automatically into `bin/` by the Makefile
  (`controller-gen`, `setup-envtest`, `golangci-lint`, `kustomize`).

## Development workflow

```bash
make manifests generate   # regenerate CRDs + deepcopy after API changes
make sync-helm-crds       # copy generated CRDs into the Helm chart (run by manifests)
make fmt vet              # format + vet
make lint                 # golangci-lint
make test                 # unit + envtest integration tests
make test-e2e             # end-to-end tests (requires a running cluster)
```

After editing anything under `api/`, always run `make manifests generate` and
commit the regenerated files. CI enforces that the generated CRDs (in both
`config/crd/bases/` and `charts/gobackup-operator/crds/`) are in sync.

## Pull requests

1. Fork and branch from `main`.
2. Keep changes focused; write tests for new behavior.
3. Ensure `make fmt vet lint test` is green locally.
4. Use clear, conventional commit messages
   (`feat:`, `fix:`, `docs:`, `refactor:`, `build:`, `ci:`, `test:`).
5. Open a PR describing the change and its motivation.

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE).
