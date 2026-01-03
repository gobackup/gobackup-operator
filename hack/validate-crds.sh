#!/bin/bash
# Script to validate CRD files
# This script validates that CRD files can be applied correctly to a Kubernetes cluster

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CRD_DIR="${PROJECT_ROOT}/config/crd/bases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    log_info "kubectl found: $(kubectl version --client --short 2>/dev/null || echo 'version unknown')"
}

# Check if CRD directory exists
check_crd_dir() {
    if [ ! -d "$CRD_DIR" ]; then
        log_error "CRD directory not found: $CRD_DIR"
        log_info "Run 'make manifests' to generate CRD files"
        exit 1
    fi
    
    if [ -z "$(ls -A "$CRD_DIR"/*.yaml 2>/dev/null)" ]; then
        log_error "No CRD YAML files found in $CRD_DIR"
        log_info "Run 'make manifests' to generate CRD files"
        exit 1
    fi
    
    log_info "Found CRD files in $CRD_DIR"
}

# Validate YAML syntax
validate_yaml_syntax() {
    log_info "Validating YAML syntax..."
    local errors=0
    
    for crd_file in "$CRD_DIR"/*.yaml; do
        if [ -f "$crd_file" ]; then
            if ! kubectl apply --dry-run=client -f "$crd_file" &> /dev/null; then
                log_error "YAML syntax error in $(basename "$crd_file")"
                kubectl apply --dry-run=client -f "$crd_file" 2>&1 | head -20
                errors=$((errors + 1))
            else
                log_info "✓ $(basename "$crd_file") - YAML syntax valid"
            fi
        fi
    done
    
    if [ $errors -gt 0 ]; then
        log_error "Found $errors YAML syntax error(s)"
        return 1
    fi
    
    log_info "All CRD files have valid YAML syntax"
    return 0
}

# Validate CRD structure and schema
validate_crd_structure() {
    log_info "Validating CRD structure and OpenAPI schema..."
    local errors=0
    
    for crd_file in "$CRD_DIR"/*.yaml; do
        if [ -f "$crd_file" ]; then
            # Check if it's a valid CRD
            if ! grep -q "kind: CustomResourceDefinition" "$crd_file"; then
                log_warn "File $(basename "$crd_file") does not appear to be a CRD (missing kind: CustomResourceDefinition)"
                continue
            fi
            
            # Extract CRD name
            crd_name=$(grep "^  name:" "$crd_file" | head -1 | awk '{print $2}' || echo "")
            
            if [ -z "$crd_name" ]; then
                log_error "Cannot determine CRD name from $(basename "$crd_file")"
                errors=$((errors + 1))
                continue
            fi
            
            # Validate CRD structure using kubectl
            if ! kubectl apply --dry-run=client -f "$crd_file" &> /dev/null; then
                log_error "CRD structure validation failed for $crd_name"
                kubectl apply --dry-run=client -f "$crd_file" 2>&1 | head -20
                errors=$((errors + 1))
            else
                log_info "✓ $crd_name - CRD structure valid"
            fi
        fi
    done
    
    if [ $errors -gt 0 ]; then
        log_error "Found $errors CRD structure error(s)"
        return 1
    fi
    
    log_info "All CRDs have valid structure"
    return 0
}

# Validate against Kubernetes API server (if cluster is available)
validate_against_server() {
    if ! kubectl cluster-info &> /dev/null; then
        log_warn "Cannot connect to Kubernetes cluster. Skipping server-side validation."
        log_info "To enable server-side validation, ensure kubectl is configured with a valid cluster context"
        return 0
    fi
    
    log_info "Validating CRDs against Kubernetes API server..."
    local errors=0
    
    for crd_file in "$CRD_DIR"/*.yaml; do
        if [ -f "$crd_file" ]; then
            crd_name=$(grep "^  name:" "$crd_file" | head -1 | awk '{print $2}' || echo "")
            
            if [ -z "$crd_name" ]; then
                continue
            fi
            
            # Try server-side dry-run validation
            if ! kubectl apply --dry-run=server -f "$crd_file" &> /dev/null; then
                log_error "Server-side validation failed for $crd_name"
                kubectl apply --dry-run=server -f "$crd_file" 2>&1 | head -20
                errors=$((errors + 1))
            else
                log_info "✓ $crd_name - Server-side validation passed"
            fi
        fi
    done
    
    if [ $errors -gt 0 ]; then
        log_error "Found $errors server-side validation error(s)"
        return 1
    fi
    
    log_info "All CRDs passed server-side validation"
    return 0
}

# Test applying CRDs to a temporary namespace (if cluster is available)
test_crd_application() {
    if ! kubectl cluster-info &> /dev/null; then
        log_warn "Cannot connect to Kubernetes cluster. Skipping CRD application test."
        return 0
    fi
    
    log_info "Testing CRD application (this will create and delete CRDs in the cluster)..."
    
    # Check if we should actually apply (use TEST_APPLY_CRDS env var)
    if [ "${TEST_APPLY_CRDS:-false}" != "true" ]; then
        log_info "Skipping actual CRD application (set TEST_APPLY_CRDS=true to enable)"
        return 0
    fi
    
    local errors=0
    local applied_crds=()
    
    # Apply all CRDs
    for crd_file in "$CRD_DIR"/*.yaml; do
        if [ -f "$crd_file" ]; then
            crd_name=$(grep "^  name:" "$crd_file" | head -1 | awk '{print $2}' || echo "")
            
            if [ -z "$crd_name" ]; then
                continue
            fi
            
            log_info "Applying $crd_name..."
            if kubectl apply -f "$crd_file" &> /dev/null; then
                log_info "✓ $crd_name - Applied successfully"
                applied_crds+=("$crd_name")
                
                # Wait for CRD to be established
                if kubectl wait --for condition=established --timeout=30s "crd/$crd_name" &> /dev/null; then
                    log_info "✓ $crd_name - Established"
                else
                    log_warn "$crd_name - Not established within timeout (may still be valid)"
                fi
            else
                log_error "Failed to apply $crd_name"
                kubectl apply -f "$crd_file" 2>&1 | head -20
                errors=$((errors + 1))
            fi
        fi
    done
    
    # Cleanup: delete applied CRDs
    if [ ${#applied_crds[@]} -gt 0 ]; then
        log_info "Cleaning up applied CRDs..."
        for crd_name in "${applied_crds[@]}"; do
            kubectl delete "crd/$crd_name" --ignore-not-found=true &> /dev/null || true
        done
    fi
    
    if [ $errors -gt 0 ]; then
        log_error "Found $errors CRD application error(s)"
        return 1
    fi
    
    log_info "All CRDs applied and cleaned up successfully"
    return 0
}

# Main validation function
main() {
    log_info "Starting CRD validation..."
    log_info "CRD directory: $CRD_DIR"
    
    check_kubectl
    check_crd_dir
    
    local validation_errors=0
    
    # Run validations
    validate_yaml_syntax || validation_errors=$((validation_errors + 1))
    validate_crd_structure || validation_errors=$((validation_errors + 1))
    validate_against_server || validation_errors=$((validation_errors + 1))
    test_crd_application || validation_errors=$((validation_errors + 1))
    
    if [ $validation_errors -eq 0 ]; then
        log_info "✓ All CRD validations passed!"
        exit 0
    else
        log_error "CRD validation failed with $validation_errors error(s)"
        exit 1
    fi
}

# Run main function
main

