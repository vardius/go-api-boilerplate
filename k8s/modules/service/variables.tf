variable "namespace" {
  type = string
  description = "namespace"
}

variable "labels" {
  type = map(string)
  description = "pod labels"
  default = {}
}

variable "annotations" {
  type = map(string)
  description = "pod annotations"
  default = {}
}

variable "persistence" {
  type = list(object({
    storageClass = string
    claim        = string
  }))
  description = "pod persistence volume settings"
  default = []
}

variable "imagePullSecretName" {
  type = string
  description = "image pull secret name"
}

variable "envSecretName" {
  type = string
  description = "image pull secret name"
}

variable "name" {
  type = string
  description = "service full name override"
}

variable "database" {
  type = string
  description = "database name"
}

variable "renderedValues" {
  type = string
  description = "service rendered values from template"
}
