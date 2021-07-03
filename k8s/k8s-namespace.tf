resource "kubernetes_namespace" "main" {
  metadata {
    name        = local.app
    labels      = local.labels
    annotations = local.annotations
  }
}
