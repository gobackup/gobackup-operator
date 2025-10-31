#!/bin/bash
# End-to-end test script for gobackup-operator
# Following best practices from operators like prometheus-operator

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test namespace
TEST_NS="gobackup-e2e-test-$(date +%s)"

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

trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    if ! command -v kind &> /dev/null; then
        log_warn "kind is not installed. E2E tests will use current kubectl context."
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
    kubectl create namespace "$TEST_NS"
    
    # Ensure CRDs are installed
    log_info "Installing CRDs..."
    kubectl apply -f "$PROJECT_ROOT/config/crd/bases" || true
    
    # Wait for CRDs to be ready
    log_info "Waiting for CRDs to be ready..."
    kubectl wait --for condition=established --timeout=60s \
        crd/postgresqls.gobackup.io \
        crd/s3s.gobackup.io \
        crd/backups.gobackup.io || true
}

# Deploy operator
deploy_operator() {
    log_info "Deploying operator..."
    
    # Check if operator is already deployed
    if kubectl get deployment -n gobackup-operator-system gobackup-operator-controller-manager &> /dev/null; then
        log_info "Operator already deployed, skipping..."
        return
    fi
    
    # Deploy operator
    cd "$PROJECT_ROOT"
    make deploy IMG=controller:latest
    
    # Wait for operator to be ready
    log_info "Waiting for operator to be ready..."
    kubectl wait --for=condition=available --timeout=120s \
        deployment/gobackup-operator-controller-manager \
        -n gobackup-operator-system || {
        log_error "Operator failed to become ready"
        kubectl logs -n gobackup-operator-system -l control-plane=controller-manager --tail=50
        exit 1
    }
}

# Test immediate backup
test_immediate_backup() {
    log_info "Testing immediate backup..."
    
    # Create PostgreSQL resource
    log_info "Creating PostgreSQL resource..."
    cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: PostgreSQL
metadata:
  name: test-postgres
  namespace: $TEST_NS
spec:
  host: "postgres.example.com"
  port: 5432
  username: "testuser"
  password: "testpass"
  database: "testdb"
EOF

    # Create S3 resource
    log_info "Creating S3 resource..."
    cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: S3
metadata:
  name: test-s3
  namespace: $TEST_NS
spec:
  bucket: "test-bucket"
  region: "us-east-1"
  accessKeyID: "test-key"
  secretAccessKey: "test-secret"
EOF

    # Wait a bit for resources to settle
    sleep 2

    # Create Backup resource
    log_info "Creating Backup resource..."
    cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: test-backup-immediate
  namespace: $TEST_NS
spec:
  databaseRefs:
    - apiGroup: gobackup.io
      type: PostgreSQL
      name: test-postgres
  storageRefs:
    - apiGroup: gobackup.io
      type: S3
      name: test-s3
      keep: 5
      timeout: 300
  compressWith:
    type: gzip
EOF

    # Wait for secret to be created
    log_info "Waiting for secret to be created..."
    timeout 30 kubectl wait --for=condition=ready secret/test-backup-immediate -n "$TEST_NS" || {
        log_error "Secret was not created"
        kubectl get secrets -n "$TEST_NS"
        exit 1
    }

    # Wait for job to be created
    log_info "Waiting for job to be created..."
    timeout 30 kubectl wait --for=condition=ready job/test-backup-immediate -n "$TEST_NS" || {
        log_error "Job was not created"
        kubectl get jobs -n "$TEST_NS"
        exit 1
    }

    # Verify job spec
    log_info "Verifying job spec..."
    JOB_IMAGE=$(kubectl get job test-backup-immediate -n "$TEST_NS" -o jsonpath='{.spec.template.spec.containers[0].image}')
    if [ "$JOB_IMAGE" != "huacnlee/gobackup" ]; then
        log_error "Job image is incorrect: $JOB_IMAGE"
        exit 1
    fi

    log_info "Immediate backup test passed ✓"
}

# Test scheduled backup
test_scheduled_backup() {
    log_info "Testing scheduled backup..."
    
    # Create Backup with schedule
    cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: test-backup-scheduled
  namespace: $TEST_NS
spec:
  databaseRefs:
    - apiGroup: gobackup.io
      type: PostgreSQL
      name: test-postgres
  storageRefs:
    - apiGroup: gobackup.io
      type: S3
      name: test-s3
      keep: 10
      timeout: 600
  compressWith:
    type: gzip
  schedule:
    cron: "0 */6 * * *"
    successfulJobsHistoryLimit: 3
    failedJobsHistoryLimit: 1
EOF

    # Wait for secret to be created
    log_info "Waiting for secret to be created..."
    timeout 30 kubectl wait --for=condition=ready secret/test-backup-scheduled -n "$TEST_NS" || {
        log_error "Secret was not created"
        kubectl get secrets -n "$TEST_NS"
        exit 1
    }

    # Wait for cronjob to be created
    log_info "Waiting for cronjob to be created..."
    timeout 30 kubectl wait --for=condition=ready cronjob/test-backup-scheduled -n "$TEST_NS" || {
        log_error "CronJob was not created"
        kubectl get cronjobs -n "$TEST_NS"
        exit 1
    }

    # Verify cronjob spec
    log_info "Verifying cronjob spec..."
    CRON_SCHEDULE=$(kubectl get cronjob test-backup-scheduled -n "$TEST_NS" -o jsonpath='{.spec.schedule}')
    if [ "$CRON_SCHEDULE" != "0 */6 * * *" ]; then
        log_error "CronJob schedule is incorrect: $CRON_SCHEDULE"
        exit 1
    fi

    log_info "Scheduled backup test passed ✓"
}

# Main test execution
main() {
    log_info "Starting E2E tests for gobackup-operator"
    
    check_prerequisites
    setup_test_env
    deploy_operator
    
    test_immediate_backup
    test_scheduled_backup
    
    log_info "All E2E tests passed ✓"
}

# Run main function
main

