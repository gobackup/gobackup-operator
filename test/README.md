# Testing Guide for gobackup-operator

This document outlines the testing strategy for gobackup-operator, following best practices from well-known Kubernetes operators like prometheus-operator, ArgoCD operator, and etcd-operator.

## Testing Strategy

The testing approach follows a three-tier strategy:

1. **Unit Tests**: Fast, isolated tests that don't require a Kubernetes API server
2. **Integration Tests**: Tests using envtest that require a Kubernetes API server but no actual cluster
3. **End-to-End (E2E) Tests**: Tests that run against a real Kubernetes cluster

## Prerequisites

- Go 1.21+
- kubectl configured to connect to a Kubernetes cluster (for E2E tests)
- [kind](https://kind.sigs.k8s.io/) (optional, for local E2E testing)
- Docker (for building operator images)

## Running Tests

### Unit Tests

Fast unit tests that test individual functions:

```bash
make test-unit
```

### Integration Tests

Integration tests using envtest (requires test binaries):

```bash
make test-integration
```

### All Tests (Unit + Integration)

Run all tests including integration tests:

```bash
make test
```

### End-to-End Tests

E2E tests require a running Kubernetes cluster:

```bash
# Using an existing cluster
make test-e2e

# Or using kind (local cluster)
make kind-run
make test-e2e
```

### Test Coverage

Generate a test coverage report:

```bash
make test-coverage
```

This generates `cover.html` which you can open in a browser to see coverage details.

## Test Structure

### Unit Tests

Located in package directories with `_test.go` suffix:
- `pkg/k8sutil/*_test.go` - Utility function tests

### Integration Tests

Located in `internal/controller/`:
- `suite_test.go` - Test suite setup with envtest
- `backup_controller_test.go` - Controller reconciliation tests

### E2E Tests

Located in `test/e2e/`:
- `e2e_test.sh` - End-to-end test script

## Writing Tests

### Integration Test Example

Integration tests use Ginkgo and Gomega (BDD-style testing):

```go
var _ = Describe("Backup Controller", func() {
    var (
        ctx            context.Context
        testNamespace string
    )

    BeforeEach(func() {
        ctx = context.Background()
        testNamespace = "test-" + time.Now().Format("20060102-150405")
        // Create test namespace...
    })

    It("Should create a Job for immediate backup", func() {
        // Test implementation
    })
})
```

### Test Best Practices

1. **Isolation**: Each test should be independent and not rely on other tests
2. **Cleanup**: Always clean up resources in `AfterEach` blocks
3. **Timeouts**: Use `Eventually` with appropriate timeouts for async operations
4. **Fixtures**: Use helper functions to create test resources
5. **Assertions**: Use descriptive `Expect` statements with clear error messages

## Test Coverage Goals

- **Minimum**: 70% overall coverage
- **Critical paths**: 90%+ coverage for controller reconciliation logic
- **Target**: 80%+ overall coverage

## Continuous Integration

Tests are designed to run in CI/CD pipelines:

```bash
# CI pipeline example
make test
make lint
make build
```

## Troubleshooting

### envtest Issues

If envtest tests fail, ensure binaries are downloaded:

```bash
make envtest
```

### E2E Test Failures

If E2E tests fail:

1. Check cluster connectivity: `kubectl cluster-info`
2. Verify CRDs are installed: `kubectl get crd | grep gobackup`
3. Check operator logs: `kubectl logs -n gobackup-operator-system -l control-plane=controller-manager`

## Additional Resources

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [Kubebuilder Testing](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html)
- [envtest Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)

