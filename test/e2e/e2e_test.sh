#!/bin/bash
# End-to-end test script for gobackup-operator
# Following best practices from operators like prometheus-operator

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MANIFESTS_DIR="$PROJECT_ROOT/test/e2e/manifests"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
TEST_NS="gobackup-e2e-test"
OPERATOR_NS="gobackup-operator-system"
WAIT_TIMEOUT="180s"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up test resources..."
    kubectl delete namespace "$TEST_NS" --ignore-not-found=true || true
}

# Trap cleanup on exit unless NO_CLEANUP is set
if [ -z "$NO_CLEANUP" ]; then
    trap cleanup EXIT
fi

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."
    
    # Create test namespace
    kubectl create namespace "$TEST_NS" --dry-run=client -o yaml | kubectl apply -f -
    
    # Ensure CRDs are installed
    log_info "Installing CRDs..."
    kubectl apply -f "$PROJECT_ROOT/config/crd/bases"
    
    # Wait for CRDs to be ready
    log_info "Waiting for CRDs to be ready..."
    kubectl wait --for condition=established --timeout=60s \
        crd/backups.gobackup.io \
        crd/databases.gobackup.io \
        crd/storages.gobackup.io || true
}

# Deploy operator
deploy_operator() {
    log_info "Deploying operator..."
    
    # Build and load image if using kind
    if kubectl config current-context | grep -q "^kind-"; then
        CLUSTER_NAME=$(kubectl config current-context | sed 's/^kind-//')
        log_info "Kind cluster detected: $CLUSTER_NAME. Building and loading image..."
        make docker-build IMG=payamqorbanpour/gobackup-operator:dev
        kind load docker-image payamqorbanpour/gobackup-operator:dev --name "$CLUSTER_NAME"
        make deploy IMG=payamqorbanpour/gobackup-operator:dev
    else
        make deploy IMG=controller:latest
    fi
    
    # Force IfNotPresent for Kind to use local image
    log_info "Fixing image pull policy for Kind..."
    kubectl patch deployment gobackup-operator-controller-manager -n "$OPERATOR_NS" -p '{"spec":{"template":{"spec":{"containers":[{"name":"manager","imagePullPolicy":"IfNotPresent"}]}}}}' || true

    # Wait for operator to be ready
    log_info "Waiting for operator to be ready in $OPERATOR_NS..."
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" \
        deployment/gobackup-operator-controller-manager \
        -n "$OPERATOR_NS" || {
        log_error "Operator failed to become ready"
        kubectl get pods -n "$OPERATOR_NS"
        kubectl logs -n "$OPERATOR_NS" -l control-plane=controller-manager -c manager --tail=100
        exit 1
    }
}

# Deploy infrastructure (Minio, FTP, SFTP, WebDAV)
setup_infra() {
    log_info "Setting up infrastructure (Minio, FTP, SFTP, WebDAV)..."
    kubectl apply -f "$MANIFESTS_DIR/infra/minio.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/infra/ftp.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/infra/sftp.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/infra/webdav.yaml" -n "$TEST_NS"
    
    log_info "Waiting for infrastructure to be ready..."
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" deployment/minio -n "$TEST_NS"
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" deployment/ftp -n "$TEST_NS"
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" deployment/sftp -n "$TEST_NS"
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" deployment/webdav -n "$TEST_NS"
    
    log_info "Applying Storage CRs..."
    kubectl apply -f "$MANIFESTS_DIR/crs/storage-minio.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/crs/storage-local.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/crs/storage-ftp.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/crs/storage-sftp.yaml" -n "$TEST_NS"
    kubectl apply -f "$MANIFESTS_DIR/crs/storage-webdav.yaml" -n "$TEST_NS"
}

# Test a specific database
test_database() {
    local db_type=$1
    log_info ">>> Testing $db_type backup..."
    
    # 1. Deploy DB Infra
    log_info "Deploying $db_type infrastructure..."
    kubectl apply -f "$MANIFESTS_DIR/infra/${db_type}.yaml" -n "$TEST_NS"
    
    # Special wait for different DB names in deployment
    log_info "Waiting for $db_type to be ready..."
    local deploy_name=$db_type
    if [ "$db_type" == "postgresql" ]; then deploy_name="postgres"; fi
    
    kubectl wait --for=condition=available --timeout="$WAIT_TIMEOUT" deployment/$deploy_name -n "$TEST_NS"

    # 2. Apply Database and Backup CRs
    log_info "Applying CRs for $db_type..."
    kubectl apply -f "$MANIFESTS_DIR/crs/db-backup-${db_type}.yaml" -n "$TEST_NS"
    
    # Wait a bit for controller to process
    sleep 5
    
    # 3. Verify Secret creation
    log_info "Verifying Secret creation..."
    timeout 60 bash -c "until kubectl get secret backup-${db_type} -n $TEST_NS; do sleep 2; done" || {
        log_error "Secret backup-${db_type} was not created"
        log_info "Dumping debug information..."
        log_info "=== Backup Resource ==="
        kubectl get backup backup-${db_type} -n "$TEST_NS" -o yaml
        log_info "=== Backup Status ==="
        kubectl describe backup backup-${db_type} -n "$TEST_NS"
        log_info "=== Database Resource ==="
        kubectl get database ${db_type} -n "$TEST_NS" -o yaml
        log_info "=== Storage Resource ==="
        kubectl get storage minio -n "$TEST_NS" -o yaml
        log_info "=== Controller Logs ==="
        kubectl logs -n "$OPERATOR_NS" -l control-plane=controller-manager -c manager --tail=100
        log_info "=== Events ==="
        kubectl get events -n "$TEST_NS" --sort-by='.lastTimestamp'
        exit 1
    }
    
    # 4. Verify Job creation
    log_info "Verifying Job creation..."
    timeout 60 bash -c "until kubectl get job backup-${db_type} -n $TEST_NS; do sleep 2; done" || {
        log_error "Job backup-${db_type} was not created"
        kubectl describe backup backup-${db_type} -n "$TEST_NS"
        exit 1
    }
    
    # 5. Verify Job Image
    JOB_IMAGE=$(kubectl get job backup-${db_type} -n "$TEST_NS" -o jsonpath='{.spec.template.spec.containers[0].image}')
    if [[ "$JOB_IMAGE" != *"gobackup"* ]]; then
        log_error "Job image is incorrect: $JOB_IMAGE"
        exit 1
    fi

    log_info "$db_type test passed ✓"
}

# Main test execution
main() {
    log_info "Starting Comprehensive E2E tests for gobackup-operator"
    
    check_prerequisites
    setup_test_env
    deploy_operator
    setup_infra
    
    DATABASES=("postgresql" "mysql" "mariadb" "redis" "mongodb" "mssql" "influxdb" "etcd")
    
    for db in "${DATABASES[@]}"; do
        test_database "$db"
    done
    
    log_info "========================================"
    log_info "🎉 All E2E tests passed successfully! 🎉"
    log_info "========================================"
}

# Run main function
main

