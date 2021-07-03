output "namespace" {
  value = var.namespace
  description = "namespace"
}

output "labels" {
  value = var.labels
  description = "pod labels"
}

output "persistence" {
  value = var.persistence
  description = "pod persistence volume settings"
}

output "annotations" {
  value = var.annotations
  description = "pod annotations"
}

output "imagePullSecretName" {
  value = var.imagePullSecretName
  description = "image pull secret name"
}

output "envSecretName" {
  value = var.envSecretName
  description = "image pull secret name"
}

output "name" {
  value = var.name
  description = "service full name override"
}

output "databaseName" {
  value = var.database
  description = "database name"
}

output "renderedValues" {
  value = var.renderedValues
  description = "service rendered values from template"
}