resource "helm_release" "maildev" {
  name      = "maildev"
  chart     = "./helm/charts/microservice"
  namespace = kubernetes_namespace.main.metadata[0].name

  values = [
    <<-EOF
    podLabels:
      ${chomp(indent(2, yamlencode(local.labels)))}
    podAnnotations:
      ${chomp(indent(2, yamlencode(merge(local.annotations, {
        "go-api-boilerplate.local/role"    = "api"
        "go-api-boilerplate.local/project" = "maildev-api-go"
      }))))}

    nameOverride: maildev
    image:
      repository: maildev/maildev
      tag: 2.0.0-beta3
      pullPolicy: IfNotPresent
    service:
      ports:
        - name: maildev
          internalPort: 1025
          externalPort: 1025
        - name: ui
          internalPort: 1080
          externalPort: 1080
    env:
      - name: MAILDEV_INCOMING_USER
        valueFrom:
          secretKeyRef:
            key: MAILER_USER
            name: ${kubernetes_secret.config.metadata.0.name}
      - name: MAILDEV_INCOMING_PASS
        valueFrom:
          secretKeyRef:
            key: MAILER_PASSWORD
            name: ${kubernetes_secret.config.metadata.0.name}
    EOF
  ]
}
