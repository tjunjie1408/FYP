# Cluster-agnostic bootstrap: installs exactly two things (Cilium, Argo CD)
# plus the root Application. Everything else is GitOps (ADR-0001).
# Providers (helm, kubectl) are configured by the calling env, never here.

variable "api_server_host" {
  type        = string
  description = "API server address reachable from pods (kind: '<name>-control-plane'; GKE: endpoint IP)."
}

variable "api_server_port" {
  type    = number
  default = 6443
}

variable "cilium_version" {
  type    = string
  default = "1.19.4"
}

variable "argocd_chart_version" {
  type        = string
  description = "argo-cd helm chart version (chart 9.x = Argo CD 3.4.x)."
  default     = "9.5.0"
}

variable "gitops_repo_url" {
  type        = string
  description = "Git URL of THIS repo (your fork) that Argo CD syncs from."
}

variable "gitops_revision" {
  type    = string
  default = "HEAD"
}

variable "cilium_values_file" {
  type        = string
  description = "Path to a Cilium values template; defaults to the kind profile."
  default     = ""
}
