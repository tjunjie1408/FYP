# OPTIONAL cloud target — written, not run by default (costs money).
# Promote to primary demo target if the laptop can't give WSL2 16 GB
# (decide by end of M1, not on demo day). T4 spot ~ $0.11-0.16/h.
resource "google_container_cluster" "this" {
  name     = var.cluster_name
  location = var.zone
  project  = var.project_id

  remove_default_node_pool = true
  initial_node_count       = 1
  deletion_protection      = false

  # Default dataplane kept; note GKE Dataplane V2 *is* Cilium — if you enable
  # it (datapath_provider = "ADVANCED_DATAPATH"), skip the helm Cilium install
  # and keep only the CiliumNetworkPolicies (ADR-0002).
}

resource "google_container_node_pool" "system" {
  name     = "system"
  cluster  = google_container_cluster.this.name
  location = var.zone
  project  = var.project_id

  node_count = 1
  node_config {
    machine_type = "e2-standard-4"
    disk_size_gb = 50
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}

resource "google_container_node_pool" "gpu" {
  name     = "gpu-t4-spot"
  cluster  = google_container_cluster.this.name
  location = var.zone
  project  = var.project_id

  initial_node_count = 0
  autoscaling {
    min_node_count = 0
    max_node_count = 1
  }

  node_config {
    machine_type = "n1-standard-4"
    spot         = true
    disk_size_gb = 50
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    guest_accelerator {
      type  = "nvidia-tesla-t4"
      count = 1
    }

    labels = {
      "nvidia.com/gpu.present" = "true"
    }
    taint {
      key    = "nvidia.com/gpu"
      value  = "present"
      effect = "NO_SCHEDULE"
    }
  }
}
