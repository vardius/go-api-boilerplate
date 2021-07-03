# https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html
# https://github.com/jetstack/cert-manager/tree/master/deploy/charts/cert-manager
resource "helm_release" "main" {
  name       = "cert-manager"
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  version    = local.cert-manager.version
  namespace  = local.cert-manager.namespace

  values = [
    <<-EOF
    podLabels:
      ${chomp(indent(2, yamlencode(local.labels)))}
    podAnnotations:
      ${chomp(indent(2, yamlencode(local.annotations)))}

    ingressShim:
      defaultIssuerName: selfsigned
      defaultIssuerKind: Issuer
    EOF
  ]
}
