<p align="center">
  <img src="https://github.com/user-attachments/assets/eb9f7270-9250-4d41-915b-c2debc873741" width="250" />
</p>

</p>

# gobackup-operator

**Please note:** This project is currently under active development.

Handling backup from various storages.

## Description
A Kubernetes operator for backing up various storages, including Etcd, based on [gobackup](https://github.com/gobackup/gobackup).

## Getting Started

### Prerequisites
- Golang
- Docker
- Kubectl
- Access to a Kubernetes cluster

## Structure

```
gobackup-operator/
├── .github/               # CI/CD workflows (GitHub Actions)
├── api/                   # API definitions (CustomResourceDefinitions)
├── build/                 # Build artifacts
├── cmd/                   # Entry point for the operator
├── config/
│   ├── crd/               # Custom Resource Definitions (CRDs)
│   ├── default/           # Default manifests (e.g., manager deployment, cluster roles, RBAC)
│   ├── manager/           # Operator deployment manifests (e.g., Deployment.yaml, Service.yaml)
│   ├── rbac/              # RBAC permissions (e.g., ClusterRole.yaml, Role.yaml, RoleBinding.yaml)
│   ├── samples/           # Example custom resources (CRs) to test your operator
├── internal/
│   ├──controller/           # Controller logic
├── pkg/                   # internal utils
├── Makefile               # Automation scripts (build, deploy, test)
├── PROJECT                # Operator SDK/Kubebuilder metadata
├── README.md              # Documentation
```

### To Deploy on the cluster

**Install the CRDs into the cluster:**

```sh
make install
```

### To Test the Operator on the cluster

**Create instances of database and storage**

```sh
kubectl apply -k example/gobackup-opetator-database/*
kubectl apply -k example/gobackup-opetator-storage/*
```

**Create a test database deployment**

```sh
kubectl apply -k example/gobackup-opetator-postgres-deployment.yaml
```

**Create environment with required access**

```sh
kubectl apply -k example/gobackup-opetator-clusterrole.yaml
kubectl apply -k example/gobackup-opetator-clusterrolebinding.yaml
kubectl apply -k example/gobackup-opetator-namespace.yaml
kubectl apply -k example/gobackup-opetator-pvc.yaml
kubectl apply -k example/gobackup-opetator-service.yaml
kubectl apply -k example/gobackup-opetator-serviceaccount.yaml
```

**Deploy the Operator**

```sh
kubectl apply -k example/gobackup-opetator-deployment.yaml
```

**Create model and backup instances**

This will trigger the Operator to run a backup command

```sh
kubectl apply -k example/gobackup-opetator/gobackup-opetator-backupmodel.yaml
kubectl apply -k example/gobackup-opetator/gobackup-opetator-backup.yaml
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k example/*
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

## Contributing

Just create a new branch (feature-{branch-name}) and push.

When you finish your work, please send a PR.

## License

MIT
