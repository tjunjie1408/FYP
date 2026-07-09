variable "cluster_name" {
  type    = string
  default = "fyp"
}

variable "node_image" {
  type        = string
  description = "kind node image. KServe 0.18 requires k8s >= 1.32; pinned, don't ride latest."
  default     = "kindest/node:v1.33.1"
}

variable "kubeconfig_path" {
  type        = string
  description = "Where the kind provider writes the kubeconfig for this cluster."
  default     = ""
}
