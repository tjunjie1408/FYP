# Operator Makefile — trimmed Kubebuilder-style. Tools are invoked via
# `go run` with pinned versions, so no local tool installs are required.

CONTROLLER_GEN := go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
# Windows note: `go run .../setup-envtest` trips the UAC installer heuristic
# (exe names starting with "setup" require elevation). On Windows instead:
#   go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.21
#   rename it (e.g. envtest-tool.exe), then: envtest-tool use 1.33.0 -p path
SETUP_ENVTEST  := go run sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.21
ENVTEST_K8S_VERSION := 1.33.0

IMG ?= fyp/ml-operator:dev

.PHONY: all manifests generate fmt vet test build run docker-build

all: build

# Explicit package paths (not ./...) — works on both Linux and Windows.
GEN_PATHS := paths=./api/v1alpha1 paths=./internal/controller

## manifests: generate CRDs and RBAC into config/
manifests:
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd $(GEN_PATHS) \
		output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac

## generate: generate zz_generated.deepcopy.go
generate:
	$(CONTROLLER_GEN) object paths=./api/v1alpha1

fmt:
	go fmt ./...

vet:
	go vet ./...

## test: unit + envtest suite (downloads kube-apiserver/etcd test binaries)
test: manifests generate fmt vet
	KUBEBUILDER_ASSETS="$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir bin -p path)" \
		go test ./... -coverprofile cover.out

build: manifests generate fmt vet
	go build -o bin/manager cmd/main.go

## run: run the operator locally against the current kubeconfig context.
## Possible because there are no webhooks — the fastest dev loop.
run: manifests generate fmt vet
	go run ./cmd/main.go

docker-build:
	docker build -t $(IMG) .
