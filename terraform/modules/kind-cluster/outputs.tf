output "kubeconfig" {
  value     = kind_cluster.this.kubeconfig
  sensitive = true
}

output "endpoint" {
  value = kind_cluster.this.endpoint
}

output "client_certificate" {
  value     = kind_cluster.this.client_certificate
  sensitive = true
}

output "client_key" {
  value     = kind_cluster.this.client_key
  sensitive = true
}

output "cluster_ca_certificate" {
  value     = kind_cluster.this.cluster_ca_certificate
  sensitive = true
}

output "cluster_name" {
  value = kind_cluster.this.name
}

# Cilium's kubeProxyReplacement needs the API server address as seen from
# inside the cluster network: the control-plane node container.
output "api_server_host" {
  value = "${kind_cluster.this.name}-control-plane"
}
