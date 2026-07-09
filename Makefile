# Root Makefile — cluster lifecycle, images, demo, report.
# Run from WSL2 / Git Bash. Operator-specific targets live in operator/Makefile.

GITOPS_REPO_URL ?= https://github.com/tjunjie1408/FYP
TF_LOCAL        := terraform -chdir=terraform/envs/local
KIND_CLUSTER    ?= fyp

IMG_OPERATOR  ?= fyp/ml-operator:dev
IMG_TRAINER   ?= fyp/trainer:dev
IMG_SERVER    ?= fyp/mnist-server:dev
IMG_FAKE_DCGM ?= fyp/fake-dcgm-exporter:dev

TF_VM        := terraform -chdir=terraform/envs/vm
GCP_PROJECT  ?= CHANGE-ME-gcp-project

.PHONY: up down images load demo report test smoke fmt dev-slice dev-slice-down vm-up vm-down

## up: terraform apply — kind + Cilium + Argo CD + root app
up:
	$(TF_LOCAL) init -upgrade
	$(TF_LOCAL) apply -auto-approve -var gitops_repo_url=$(GITOPS_REPO_URL)

## down: destroy the local cluster
down:
	$(TF_LOCAL) destroy -auto-approve -var gitops_repo_url=$(GITOPS_REPO_URL)

## dev-slice: light local cluster for operator dev (kind + Argo WF + MinIO, ~2GB)
dev-slice:
	scripts/dev-slice.sh

dev-slice-down:
	kind delete cluster --name fyp-dev

## vm-up: provision the 32GB cloud VM that runs the full stack (see docs/cloud-vm.md)
vm-up:
	$(TF_VM) init -upgrade
	$(TF_VM) apply -auto-approve -var project_id=$(GCP_PROJECT)
	$(TF_VM) output next_steps

## vm-down: destroy the cloud VM
vm-down:
	$(TF_VM) destroy -auto-approve -var project_id=$(GCP_PROJECT)

## images: build all images locally
images:
	docker build -t $(IMG_OPERATOR) operator
	docker build -t $(IMG_TRAINER) training
	docker build -t $(IMG_SERVER) serving/runtime
	docker build -t $(IMG_FAKE_DCGM) components/fake-dcgm-exporter

## load: load locally-built images into kind
load:
	kind load docker-image $(IMG_OPERATOR) --name $(KIND_CLUSTER)
	kind load docker-image $(IMG_TRAINER) --name $(KIND_CLUSTER)
	kind load docker-image $(IMG_SERVER) --name $(KIND_CLUSTER)
	kind load docker-image $(IMG_FAKE_DCGM) --name $(KIND_CLUSTER)

## demo: run the end-to-end demo (train -> queue -> serve -> scale-to-zero)
demo:
	scripts/demo.sh

## report: generate the cost/reliability report from Prometheus
report:
	python reports/cost-reliability/generate_report.py

## test: operator unit + envtest suite
test:
	$(MAKE) -C operator test

## smoke: full e2e smoke test (assumes cluster is up)
smoke:
	hack/e2e-smoke.sh

fmt:
	terraform fmt -recursive terraform
	$(MAKE) -C operator fmt
