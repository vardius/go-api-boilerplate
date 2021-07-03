resource "helm_release" "service" {
  name      = var.name
  namespace = var.namespace
  chart     = "./helm/charts/microservice"

  values = [
    <<-EOF
    nameOverride: ${var.name}

    podLabels:
      ${chomp(indent(2, yamlencode(var.labels)))}
    podAnnotations:
      ${chomp(indent(2, yamlencode(var.annotations)))}
    persistence:
      ${chomp(indent(2, yamlencode(var.persistence)))}
    imagePullSecrets: ${var.imagePullSecretName}
    envFromSecretRefs:
      - ${var.envSecretName}
    env:
      - name: MY_POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
      - name: MY_MEM_LIMIT
        valueFrom:
          resourceFieldRef:
            containerName: ${var.name}
            resource: limits.memory
      - name: MONGO_PASS
        valueFrom:
          secretKeyRef:
            key: mongodb-root-password
            name: ${var.envSecretName}
      - name: MONGO_DATABASE
        value: ${var.database}

    ${var.renderedValues}
    EOF
  ]
}
