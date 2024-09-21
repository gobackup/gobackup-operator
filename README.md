<p align="center">

<img src="https://github.com/gobackup/gobackup-operator/assets/25246658/a9e1e736-d073-4b65-a2a2-081613cc9e3b" width="500" />

</p>

# gobackup-operator

**Please note:** This project is currently under active development.

Handling backup from various storages.

## Description
A Kubernetes operator for backing up various storages, including Etcd, based on [gobackup](https://github.com/gobackup/gobackup).

## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

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
