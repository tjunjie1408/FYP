# DEFAULT environment: one-click local platform.
#   terraform init && terraform apply -var gitops_repo_url=https://github.com/tjunjie1408/FYP
# kind cluster -> Cilium -> Argo CD -> root Application; the app-of-apps
# waves in gitops/platform do the rest (~8-10 min to all-Healthy).

module "kind_cluster" {
  source       = "../../modules/kind-cluster"
  cluster_name = var.cluster_name
}

provider "helm" {
  kubernetes {
    host                   = module.kind_cluster.endpoint
    client_certificate     = module.kind_cluster.client_certificate
    client_key             = module.kind_cluster.client_key
    cluster_ca_certificate = module.kind_cluster.cluster_ca_certificate
  }
}

provider "kubectl" {
  host                   = module.kind_cluster.endpoint
  client_certificate     = module.kind_cluster.client_certificate
  client_key             = module.kind_cluster.client_key
  cluster_ca_certificate = module.kind_cluster.cluster_ca_certificate
  load_config_file       = false
}

module "bootstrap" {
  source = "../../modules/bootstrap"

  api_server_host      = module.kind_cluster.api_server_host
  cilium_version       = var.cilium_version
  argocd_chart_version = var.argocd_chart_version
  gitops_repo_url      = var.gitops_repo_url

  depends_on = [module.kind_cluster]
}
