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

**Run the operator:**

```sh
make run
```

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

## ToDo
- [ ] Add Github Actions
- [ ] Create a kubernetes secret from goabckup config file
- [ ] Config validations
- [ ] Add backup cronjob


## Releases
### gobackup-operator 0.1.0-alpha:

This release enables users to backup from PostgreSQL database and push it to S3 storage

## Contributing

Just create a new branch (feature-{branch-name}) and push.

When you finish your work, please send a PR.

## License

MIT
