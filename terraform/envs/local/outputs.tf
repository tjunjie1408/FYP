output "cluster_name" {
  value = module.kind_cluster.cluster_name
}

output "next_steps" {
  value = <<-EOT
    Cluster up. The kind kubeconfig was merged into your default kubeconfig
    (context: kind-${module.kind_cluster.cluster_name}).

    Argo CD UI:   scripts/port-forward.sh argocd   -> https://localhost:8443
    Password:     kubectl -n argocd get secret argocd-initial-admin-secret \
                    -o jsonpath='{.data.password}' | base64 -d
    Grafana:      scripts/port-forward.sh grafana  -> http://localhost:3000 (admin/admin)
    Watch sync:   kubectl -n argocd get applications -w
  EOT
}
