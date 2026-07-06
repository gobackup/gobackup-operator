# Image URL to use all building/pushing image targets
IMG ?= payamqorbanpour/gobackup-operator:dev
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# Aligned with the k8s libraries (k8s.io/api v0.35.x == Kubernetes 1.35).
ENVTEST_K8S_VERSION = 1.35.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen sync-helm-crds ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.

# Split out so controller-gen runs before the Helm sync (order matters: sync copies the freshly generated CRDs).
.PHONY: gen-crds
gen-crds: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: sync-helm-crds
sync-helm-crds: gen-crds ## Copy generated CRDs from config/crd/bases into the Helm chart.
	@mkdir -p charts/gobackup-operator/crds
	cp config/crd/bases/*.yaml charts/gobackup-operator/crds/

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run unit and integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: manifests generate fmt vet envtest ## Run only unit tests (faster, no envtest).
	go test ./pkg/... -v

.PHONY: test-integration
test-integration: manifests generate fmt vet envtest ## Run integration tests (slower, requires envtest).
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./internal/controller/... -v -coverprofile cover.out

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests (requires a Kubernetes cluster).
	@if ! kubectl cluster-info &> /dev/null; then \
		echo "Error: Cannot connect to Kubernetes cluster. Please ensure kubectl is configured."; \
		exit 1; \
	fi
	./test/e2e/e2e_test.sh

.PHONY: test-crd
test-crd: manifests ## Validate CRD files (syntax, structure, and optionally apply to cluster).
	@chmod +x hack/validate-crds.sh
	./hack/validate-crds.sh

.PHONY: test-crd-apply
test-crd-apply: manifests ## Validate CRD files by actually applying them to the cluster (requires cluster access).
	@chmod +x hack/validate-crds.sh
	TEST_APPLY_CRDS=true ./hack/validate-crds.sh

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report.
	go tool cover -html=cover.out -o cover.html
	@echo "Coverage report generated: cover.html"

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.12.2
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(GOLANGCI_LINT) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: ai-review
ai-review: ## Run AI code review on current changes (requires ARVAN_API_KEY)
	@echo "Running AI code review..."
	@./scripts/local-ai-review.sh

.PHONY: ai-review-staged
ai-review-staged: ## Run AI code review on staged changes only
	@echo "Running AI code review on staged changes..."
	@./scripts/local-ai-review.sh --staged

.PHONY: ai-review-unstaged
ai-review-unstaged: ## Run AI code review on unstaged changes only
	@echo "Running AI code review on unstaged changes..."
	@./scripts/local-ai-review.sh --unstaged

.PHONY: kind-start
kind-start: ## Start a kind cluster if it doesn't exist
	@echo "Ensuring kind cluster is running..."
	@# Check if Docker is running
	@if ! $(CONTAINER_TOOL) info > /dev/null 2>&1; then \
		echo "Error: Docker daemon is not running. Please start Docker first."; \
		exit 1; \
	fi
	@if ! command -v kind > /dev/null; then \
		echo "Error: kind is not installed. Install it from https://kind.sigs.k8s.io/docs/user/quick-start/"; \
		exit 1; \
	fi
	@if ! kind get clusters | grep -q gobackup-operator; then \
		echo "Creating gobackup-operator kind cluster..."; \
		kind create cluster --name gobackup-operator; \
	else \
		echo "Using existing gobackup-operator kind cluster"; \
	fi

.PHONY: kind-run
kind-run: fmt vet kustomize ## Run the operator on a local kind cluster
	@echo "Setting up and running the operator on a kind cluster"
	@# Check if Docker is running
	@if ! $(CONTAINER_TOOL) info > /dev/null 2>&1; then \
		echo "Error: Docker daemon is not running. Please start Docker first."; \
		exit 1; \
	fi
	@if ! command -v kind > /dev/null; then \
		echo "Error: kind is not installed. Install it from https://kind.sigs.k8s.io/docs/user/quick-start/"; \
		exit 1; \
	fi
	@if ! kind get clusters | grep -q gobackup-operator; then \
		echo "Creating gobackup-operator kind cluster..."; \
		kind create cluster --name gobackup-operator; \
	else \
		echo "Using existing gobackup-operator kind cluster"; \
	fi
	@echo "Building operator image using Dockerfile from build directory..."
	$(CONTAINER_TOOL) build -t gobackup-operator:dev -f build/Dockerfile .
	@echo "Loading image into kind cluster..."
	kind load docker-image gobackup-operator:dev --name gobackup-operator
	@echo "Installing CRDs..."
	@if [ -d "config/crd/bases" ]; then \
		echo "Using existing CRD files in config/crd/bases"; \
		$(KUBECTL) apply -f config/crd/bases; \
	else \
		echo "Warning: CRD directory not found. Trying to generate CRDs (may fail)..."; \
		$(MAKE) manifests || { \
			echo "CRD generation failed. Please run 'make manifests' separately before running this command."; \
			echo "Continuing with deployment without CRDs..."; \
		}; \
		if [ -d "config/crd/bases" ]; then \
			$(KUBECTL) apply -f config/crd/bases; \
		fi; \
	fi
	@echo "Deploying operator..."
	cd config/manager && $(KUSTOMIZE) edit set image controller=gobackup-operator:dev
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	@echo "Operator is running on kind cluster"
	@echo "Use 'kubectl get pods -n gobackup-operator-system' to verify the deployment"

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: clean
clean: ## Remove test resources created in controller.
	- $(KUBECTL) -n gobackup-operator-test delete secret gobackup-secret
	- $(KUBECTL) -n gobackup-operator-test delete job gobackup-job
	- $(KUBECTL) -n gobackup-operator-test delete deployment gobackup-operator

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} -f build/Dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' build/Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.2.1
# Matches the controller-gen.kubebuilder.io/version stamp in config/crd/bases.
CONTROLLER_TOOLS_VERSION ?= v0.19.0
# setup-envtest is released from controller-runtime branches; pin to the branch
# matching sigs.k8s.io/controller-runtime v0.23.x.
ENVTEST_VERSION ?= release-0.23

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

.PHONY: kind-delete
kind-delete: ## Delete the gobackup-operator kind cluster
	@echo "Deleting gobackup-operator kind cluster..."
	@if kind get clusters | grep -q gobackup-operator; then \
		kind delete cluster --name gobackup-operator; \
		echo "Cluster deleted."; \
	else \
		echo "No gobackup-operator cluster found."; \
	fi

.PHONY: kind-restart
kind-restart: ## Restart the operator service in the kind cluster
	@echo "Restarting gobackup-operator deployment..."
	@if kubectl get namespace gobackup-operator-system >/dev/null 2>&1; then \
		kubectl -n gobackup-operator-system rollout restart deployment gobackup-operator-controller-manager; \
		echo "Operator service restarted. Watching rollout status..."; \
		kubectl -n gobackup-operator-system rollout status deployment gobackup-operator-controller-manager; \
	else \
		echo "gobackup-operator-system namespace not found. Has the operator been deployed?"; \
	fi

.PHONY: kind-rebuild
kind-rebuild: ## Rebuild and redeploy the operator to kind cluster
	@echo "Rebuilding and redeploying operator to kind cluster..."
	$(CONTAINER_TOOL) build -t gobackup-operator:dev -f build/Dockerfile .
	kind load docker-image gobackup-operator:dev --name gobackup-operator
	kubectl -n gobackup-operator-system rollout restart deployment gobackup-operator-controller-manager
	@echo "Watching rollout status..."
	kubectl -n gobackup-operator-system rollout status deployment gobackup-operator-controller-manager

##@ Testing Environment

.PHONY: test-env-create
test-env-create: ## Create a new kind cluster for testing
	@echo "Creating test environment..."
	@if kind get clusters | grep -q gobackup-test; then \
		echo "Deleting existing gobackup-test cluster..."; \
		kind delete cluster --name gobackup-test; \
	fi
	@echo "Creating new gobackup-test cluster..."
	kind create cluster --name gobackup-test
	@echo "Creating test namespace..."
	kubectl create namespace gobackup-operator-test --dry-run=client -o yaml | kubectl apply -f -
	@echo "Test environment created successfully!"

.PHONY: test-env-delete
test-env-delete: ## Delete the test kind cluster
	@echo "Deleting test environment..."
	@if kind get clusters | grep -q gobackup-test; then \
		kind delete cluster --name gobackup-test; \
		echo "Test environment deleted successfully!"; \
	else \
		echo "No gobackup-test cluster found."; \
	fi

.PHONY: test-env-logs
test-env-logs: ## Show operator logs
	$(KUBECTL) logs -n gobackup-operator-system -l control-plane=controller-manager -c manager -f
