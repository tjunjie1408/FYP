variable "project_id" {
  type        = string
  description = "GCP project ID."
}

variable "zone" {
  type    = string
  default = "us-central1-a"
}

variable "machine_type" {
  type        = string
  description = "8 vCPU / 32 GB headroom for the full stack (kind runs inside the VM)."
  default     = "e2-standard-8"
}

variable "disk_gb" {
  type    = number
  default = 60
}

variable "use_spot" {
  type        = bool
  description = "Spot VM (~70% cheaper, can be preempted). Keep false for a graded demo; true for dev."
  default     = false
}

variable "ssh_source_ranges" {
  type        = list(string)
  description = "CIDRs allowed to SSH. Default is open; tighten to your IP/32 for safety."
  default     = ["0.0.0.0/0"]
}
