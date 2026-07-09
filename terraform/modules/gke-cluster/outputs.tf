output "endpoint" {
  value = "https://${google_container_cluster.this.endpoint}"
}

output "cluster_ca_certificate" {
  value     = google_container_cluster.this.master_auth[0].cluster_ca_certificate
  sensitive = true
}

output "name" {
  value = google_container_cluster.this.name
}
