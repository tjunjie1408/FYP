output "instance_ip" {
  value = google_compute_instance.fyp.network_interface[0].access_config[0].nat_ip
}

output "ssh" {
  description = "SSH into the VM."
  value       = "gcloud compute ssh fyp-platform --zone ${var.zone}"
}

output "ssh_with_tunnels" {
  description = "SSH + forward all UI ports to your laptop's localhost."
  value = join(" ", [
    "gcloud compute ssh fyp-platform --zone ${var.zone} --",
    "-L 8443:localhost:8443", # Argo CD
    "-L 3000:localhost:3000", # Grafana
    "-L 9001:localhost:9001", # MinIO console
    "-L 8080:localhost:8080", # KServe predict (Kourier)
  ])
}

output "next_steps" {
  value = <<-EOT
    1. Wait ~3 min for the toolchain install, then SSH in (see `ssh` output).
       Check it finished:  cat /var/log/fyp-bootstrap-done
    2. One-time docker perms:  sudo usermod -aG docker $(whoami) && newgrp docker
    3. git clone <your fork> && cd FYP
    4. make up GITOPS_REPO_URL=https://github.com/tjunjie1408/FYP
    5. From your laptop, re-SSH with tunnels (see `ssh_with_tunnels`) and open
       https://localhost:8443 (Argo CD), http://localhost:3000 (Grafana).
    6. Tear down when done:  terraform -chdir=terraform/envs/vm destroy
  EOT
}
