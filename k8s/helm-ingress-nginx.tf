# https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx
# https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/#proxy-connect-timeout
resource "helm_release" "ingress_nginx" {
  name       = "ingress-nginx"
  repository = "https://kubernetes.github.io/ingress-nginx"
  chart      = "ingress-nginx"
  version    = local.ingress-nginx.version
  namespace  = local.ingress-nginx.namespace

  values = [
    <<-EOF
    rbac:
      create: true

    tcp:
        ${chomp(indent(2, yamlencode(local.tcp)))}

    controller:
      dnsPolicy: ClusterFirst

      config:
        upstream-keepalive-timeout: "120"
        client-body-buffer-size: "10M"
        client-header-timeout: "120"
        client-body-timeout: "120"
        proxy-body-size: "25M"
        proxy-connect-timeout: "120"
        proxy-read-timeout: "120"
        proxy-send-timeout: "120"
        proxy-protocol-header-timeout: "120"

      podLabels:
        ${chomp(indent(4, yamlencode(local.labels)))}

      podAnnotations:
        ${chomp(indent(4, yamlencode(local.annotations)))}

      deploymentAnnotations:
        ${chomp(indent(4, yamlencode(local.annotations)))}

      service:
        externalTrafficPolicy: Local
        labels:
          ${chomp(indent(6, yamlencode(local.labels)))}
        annotations:
          ${chomp(indent(6, yamlencode(local.annotations)))}
    EOF
  ]
}
