resource "helm_release" "mongodb" {
  name       = "mongodb"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "mongodb"
  version    = local.mongodb.version
  namespace  = local.mongodb.namespace

  values = [
    <<-EOF
    podLabels:
      ${chomp(indent(2, yamlencode(local.labels)))}
    podAnnotations:
      ${chomp(indent(2, yamlencode(local.annotations)))}

    auth:
      existingSecret: ${kubernetes_secret.config.metadata.0.name}

    volumePermissions:
      enabled: true
    persistence:
      size: ${kubernetes_persistent_volume.mongodb.spec[0].capacity.storage}
      existingClaim: ${kubernetes_persistent_volume_claim.mongodb.metadata.0.name}
      storageClass: ${kubernetes_storage_class.mongodb.metadata.0.name}
    EOF
  ]
}

resource "kubernetes_storage_class" "mongodb" {
  metadata {
    name        = "mongodb-storage"
    labels      = local.labels
    annotations = local.annotations
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
  volume_binding_mode = "WaitForFirstConsumer"
}

resource "kubernetes_persistent_volume" "mongodb" {
  metadata {
    name        = "mongodb-pv"
    labels      = local.labels
    annotations = local.annotations
  }

  spec {
    capacity = {
      storage = "2Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      host_path {
        path = "/mnt"
      }
    }
    storage_class_name = kubernetes_storage_class.mongodb.metadata.0.name
  }
}

resource "kubernetes_persistent_volume_claim" "mongodb" {
  metadata {
    name        = "mongodb-pvc"
    namespace   = kubernetes_namespace.main.metadata[0].name
    labels      = local.labels
    annotations = local.annotations
  }

  spec {
    resources {
      requests = {
        storage = kubernetes_persistent_volume.mongodb.spec[0].capacity.storage
      }
    }
    access_modes = kubernetes_persistent_volume.mongodb.spec[0].access_modes
    volume_name = kubernetes_persistent_volume.mongodb.metadata.0.name
    storage_class_name = kubernetes_storage_class.mongodb.metadata.0.name
  }
}