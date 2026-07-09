# A single beefy VM that runs the EXACT local kind setup — "a bigger laptop".
# SSH in, clone your fork, run `make up`. Identical code path to local, so no
# new failure modes; ~$0.27/h on-demand (free with GCP's $300 credit).
# Access UIs from your laptop via SSH local port-forward (see outputs).

resource "google_compute_instance" "fyp" {
  name         = "fyp-platform"
  machine_type = var.machine_type
  zone         = var.zone
  tags         = ["fyp"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
      size  = var.disk_gb
      type  = "pd-balanced"
    }
  }

  network_interface {
    network = "default"
    access_config {} # ephemeral public IP
  }

  dynamic "scheduling" {
    for_each = var.use_spot ? [1] : []
    content {
      provisioning_model          = "SPOT"
      preemptible                 = true
      automatic_restart           = false
      instance_termination_action = "STOP"
    }
  }

  metadata = {
    startup-script = file("${path.module}/startup.sh")
  }
}

# SSH. The `default` network usually already allows SSH, but make it explicit
# (and tightenable) so this works on any project.
resource "google_compute_firewall" "ssh" {
  name    = "fyp-allow-ssh"
  network = "default"
  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  source_ranges = var.ssh_source_ranges
  target_tags   = ["fyp"]
}
