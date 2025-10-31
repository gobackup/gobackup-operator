#!/bin/bash

set -e

# This script tests the gobackup-operator by creating test CRDs

# 1. Build and deploy the operator
make install
make deploy

# Wait for the operator to be ready
echo "Waiting for the operator to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/gobackup-operator-controller-manager -n gobackup-operator-system

# 2. Create test CRDs

# Create a PostgreSQL database CR
cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: PostgreSQL
metadata:
  name: test-postgres
  namespace: default
spec:
  host: postgresql.default.svc.cluster.local
  port: 5432
  database: testdb
  username: postgres
  password: postgres
EOF

# Create an S3 storage CR
cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: S3
metadata:
  name: test-s3
  namespace: default
spec:
  type: s3
  bucket: test-bucket
  region: us-east-1
  accessKeyID: AKIAIOSFODNN7EXAMPLE
  secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  path: backups
  keep: 5
EOF

# 3. Test immediate backup
cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: test-backup-immediate
  namespace: default
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

# 4. Test scheduled backup
cat <<EOF | kubectl apply -f -
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: test-backup-scheduled
  namespace: default
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
      timeout: 300
  compressWith:
    type: gzip
  schedule:
    cron: "0 */6 * * *"
    successfulJobsHistoryLimit: 3
    failedJobsHistoryLimit: 1
EOF

# 5. Check the status of created resources
echo "Checking immediate backup job..."
kubectl get job test-backup-immediate -n default

echo "Checking scheduled backup CronJob..."
kubectl get cronjob test-backup-scheduled -n default

echo "Checking backup secrets..."
kubectl get secrets -n default | grep -E 'test-backup-(immediate|scheduled)'

echo "Test completed." 