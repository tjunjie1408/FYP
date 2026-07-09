variable "gitops_repo_url" {
  type        = string
  description = "Git URL of your fork of this repo; Argo CD syncs gitops/platform from it."
}

variable "cluster_name" {
  type    = string
  default = "fyp"
}

variable "cilium_version" {
  type    = string
  default = "1.19.4"
}

variable "argocd_chart_version" {
  type    = string
  default = "9.5.0"
}
