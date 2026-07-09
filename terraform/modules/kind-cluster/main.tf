# kind cluster, CNI-less and kube-proxy-less: Cilium (installed by the
# bootstrap module) replaces both. Nodes stay NotReady until Cilium is up —
# that is expected and why Cilium must be Terraform's job, not Argo CD's.
resource "kind_cluster" "this" {
  name            = var.cluster_name
  node_image      = var.node_image
  kubeconfig_path = var.kubeconfig_path != "" ? var.kubeconfig_path : null
  wait_for_ready  = false # nodes can't be Ready before the CNI exists

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    networking {
      disable_default_cni = true
      kube_proxy_mode     = "none"
    }

    node {
      role = "control-plane"
    }

    node {
      role = "worker"

      # The "GPU node": fake-dcgm simulates 4 T4s here; the label is also
      # ADR/report material (where a real DCGM DaemonSet would schedule).
      kubeadm_config_patches = [
        <<-EOT
        kind: JoinConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "nvidia.com/gpu.present=simulated"
        EOT
      ]

      # Host 8080/8443 -> Kourier NodePorts (pinned to 31080/31443 by the
      # knative-serving kustomize patch in gitops/components).
      extra_port_mappings {
        container_port = 31080
        host_port      = 8080
      }
      extra_port_mappings {
        container_port = 31443
        host_port      = 8443
      }
    }
  }
}
