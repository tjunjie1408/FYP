# 1. Cilium — must be Terraform's job: with disable_default_cni the nodes are
#    NotReady and nothing (including Argo CD) can schedule until a CNI exists.
resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io"
  chart      = "cilium"
  version    = var.cilium_version
  namespace  = "kube-system"

  values = [
    templatefile(
      var.cilium_values_file != "" ? var.cilium_values_file : "${path.module}/values/cilium-kind.yaml.tpl",
      {
        api_server_host = var.api_server_host
        api_server_port = var.api_server_port
      }
    )
  ]

  timeout = 600
  wait    = true
}

# 2. Argo CD — same chicken-and-egg reason. Trimmed for a laptop.
resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_chart_version
  namespace        = "argocd"
  create_namespace = true

  values = [file("${path.module}/values/argocd.yaml")]

  timeout    = 600
  wait       = true
  depends_on = [helm_release.cilium]
}

# 3. The single hand-off to GitOps: the app-of-apps root Application.
#    From here on, Argo CD owns the platform.
resource "kubectl_manifest" "root_app" {
  yaml_body = templatefile("${path.module}/root-app.yaml.tpl", {
    gitops_repo_url = var.gitops_repo_url
    gitops_revision = var.gitops_revision
  })

  depends_on = [helm_release.argocd]
}
