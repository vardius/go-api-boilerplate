provider "kubernetes" {
  config_path = pathexpand(var.kube_config)
  config_context = var.config_context
}

provider "helm" {
  kubernetes {
    config_path = pathexpand(var.kube_config)
    config_context = var.config_context
  }
}
