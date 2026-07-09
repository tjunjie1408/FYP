#!/usr/bin/env bash
# VM bootstrap: install the full toolchain so the local kind setup runs here.
# Runs as root at first boot. Marker written on success: /var/log/fyp-bootstrap-done
set -euxo pipefail

KIND_VERSION=v0.32.0
HELM_VERSION=v3.16.4
TF_VERSION=1.9.8
GO_VERSION=1.24.4

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates curl git make jq unzip

# Docker
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

# kubectl (latest stable)
KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -fsSLo /usr/local/bin/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
chmod +x /usr/local/bin/kubectl

# kind
curl -fsSLo /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"
chmod +x /usr/local/bin/kind

# helm
curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz" | tar -xz -C /tmp
mv /tmp/linux-amd64/helm /usr/local/bin/helm

# terraform
curl -fsSLo /tmp/tf.zip "https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip"
unzip -o /tmp/tf.zip -d /usr/local/bin

# go (for `make -C operator run` if you want the local operator loop)
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -xz -C /usr/local
ln -sf /usr/local/go/bin/go /usr/local/bin/go

# Raise inotify limits — kind + many controllers exhaust the defaults.
cat >/etc/sysctl.d/99-kind.conf <<EOF
fs.inotify.max_user_watches=1048576
fs.inotify.max_user_instances=8192
EOF
sysctl --system

echo "toolchain ready $(date -u)" >/var/log/fyp-bootstrap-done
