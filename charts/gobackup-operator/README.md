# GoBackup Operator Helm Chart

A Helm chart for deploying the GoBackup Operator on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

### Add the repository (if published to a Helm repository)

```bash
helm repo add gobackup https://gobackup.github.io/gobackup-operator
helm repo update
```

### Install the chart

```bash
helm install gobackup-operator gobackup/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace
```

Or install from local chart:

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace
```

### Upgrade

```bash
helm upgrade gobackup-operator gobackup/gobackup-operator \
  --namespace gobackup-operator-system
```

### Uninstall

```bash
helm uninstall gobackup-operator --namespace gobackup-operator-system
```

> **Note:** By default, CRDs are installed with the chart. To skip CRD installation, use the `--skip-crds` flag. By default, CRDs are kept when uninstalling the chart (`crds.keep=true`). To remove them:
> ```bash
> kubectl delete crd backups.gobackup.io databases.gobackup.io storages.gobackup.io
> ```

## Configuration

The following table lists the configurable parameters of the GoBackup Operator chart and their default values.

### General

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `nameOverride` | Override the chart name | `""` |
| `fullnameOverride` | Override the full name | `""` |

### Image

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Operator image repository | `ghcr.io/gobackup/gobackup-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |

### Security

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.capabilities.drop` | Drop capabilities | `["ALL"]` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |

### Leader Election

| Parameter | Description | Default |
|-----------|-------------|---------|
| `leaderElection.enabled` | Enable leader election | `true` |

### Probes

| Parameter | Description | Default |
|-----------|-------------|---------|
| `livenessProbe.httpGet.path` | Liveness probe path | `/healthz` |
| `livenessProbe.httpGet.port` | Liveness probe port | `8081` |
| `livenessProbe.initialDelaySeconds` | Initial delay | `15` |
| `livenessProbe.periodSeconds` | Period | `20` |
| `readinessProbe.httpGet.path` | Readiness probe path | `/readyz` |
| `readinessProbe.httpGet.port` | Readiness probe port | `8081` |
| `readinessProbe.initialDelaySeconds` | Initial delay | `5` |
| `readinessProbe.periodSeconds` | Period | `10` |

### Scheduling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |
| `podAnnotations` | Pod annotations | `{}` |

### CRDs

| Parameter | Description | Default |
|-----------|-------------|---------|
| `crds.keep` | Keep CRDs on uninstall | `true` |

> **Note:** CRDs are installed by default when using `helm install`. To skip CRD installation, use the `--skip-crds` flag. See [Helm documentation](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/) for more details.

### Metrics & Monitoring

| Parameter | Description | Default |
|-----------|-------------|---------|
| `metrics.enabled` | Enable metrics endpoint | `false` |
| `metrics.service.port` | Metrics service port | `8080` |
| `metrics.service.annotations` | Metrics service annotations | `{}` |
| `serviceMonitor.enabled` | Enable ServiceMonitor (requires Prometheus Operator) | `false` |
| `serviceMonitor.interval` | Scrape interval | `30s` |
| `serviceMonitor.scrapeTimeout` | Scrape timeout | `10s` |
| `serviceMonitor.labels` | ServiceMonitor labels | `{}` |
| `serviceMonitor.annotations` | ServiceMonitor annotations | `{}` |

## Examples

### Basic installation

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace
```

### With custom resources

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace \
  --set resources.limits.memory=256Mi \
  --set resources.requests.memory=128Mi
```

### With Prometheus monitoring

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace \
  --set metrics.enabled=true \
  --set serviceMonitor.enabled=true
```

### With custom image

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace \
  --set image.repository=my-registry/gobackup-operator \
  --set image.tag=v1.0.0
```

### Install without CRDs

If CRDs are already installed or you want to install them separately:

```bash
helm install gobackup-operator ./charts/gobackup-operator \
  --namespace gobackup-operator-system \
  --create-namespace \
  --skip-crds
```

## Getting Started After Installation

After installing the chart, you can start creating backup resources:

### 1. Create a Storage resource

```yaml
apiVersion: gobackup.io/v1
kind: Storage
metadata:
  name: my-s3-storage
spec:
  type: s3
  config:
    bucket: my-backup-bucket
    region: us-east-1
    access_key_id_ref:
      name: s3-credentials
      key: access-key-id
    secret_access_key_ref:
      name: s3-credentials
      key: secret-access-key
```

### 2. Create a Database resource

```yaml
apiVersion: gobackup.io/v1
kind: Database
metadata:
  name: my-postgres
spec:
  type: postgresql
  config:
    host: postgres-service
    port: 5432
    database: mydb
    username: postgres
    password: mypassword
```

### 3. Create a Backup resource

```yaml
apiVersion: gobackup.io/v1
kind: Backup
metadata:
  name: my-backup
spec:
  schedule:
    cron: "0 2 * * *"  # Daily at 2 AM
  databaseRefs:
    - name: my-postgres
      type: postgresql
  storageRefs:
    - name: my-s3-storage
      type: s3
      keep: 7
```

## Troubleshooting

### Check operator logs

```bash
kubectl logs -n gobackup-operator-system -l app.kubernetes.io/name=gobackup-operator
```

### Check CRD status

```bash
kubectl get crd | grep gobackup
```

### Check backup status

```bash
kubectl get backups -A
kubectl describe backup <backup-name>
```

## License

MIT

