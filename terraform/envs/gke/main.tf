# OPTIONAL environment — costs real money. See modules/gke-cluster/README.md.
# Same bootstrap module as local: Cilium + Argo CD + root app, then GitOps.

module "gke_cluster" {
  source     = "../../modules/gke-cluster"
  project_id = var.project_id
  zone       = var.zone
}

data "google_client_config" "default" {}

provider "helm" {
  kubernetes {
    host                   = module.gke_cluster.endpoint
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke_cluster.cluster_ca_certificate)
  }
}

provider "kubectl" {
  host                   = module.gke_cluster.endpoint
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(module.gke_cluster.cluster_ca_certificate)
  load_config_file       = false
}

module "bootstrap" {
  source = "../../modules/bootstrap"

  # On GKE the API server is the public/cluster endpoint, not a node name.
  api_server_host = replace(module.gke_cluster.endpoint, "https://", "")
  api_server_port = 443
  gitops_repo_url = var.gitops_repo_url

  depends_on = [module.gke_cluster]
}
