module "web-ui-main" {
  source = "./modules/service"

  namespace           = kubernetes_namespace.main.metadata[0].name
  labels              = local.labels
  annotations         = merge(local.annotations, {
    "go-api-boilerplate.local/role"    = "ui"
    "go-api-boilerplate.local/project" = "web-ui"
  })
  imagePullSecretName = kubernetes_secret.regcred.metadata.0.name
  envSecretName       = kubernetes_secret.config.metadata.0.name
  name                = "web-ui-main"
  database            = "web"
  renderedValues      = data.template_file.web-ui-main.rendered
}

module "auth-api-main" {
  source = "./modules/service"

  namespace           = kubernetes_namespace.main.metadata[0].name
  labels              = local.labels
  annotations         = merge(local.annotations, {
    "go-api-boilerplate.local/role"    = "api"
    "go-api-boilerplate.local/project" = "auth-api-go"
  })
  imagePullSecretName = kubernetes_secret.regcred.metadata.0.name
  envSecretName       = kubernetes_secret.config.metadata.0.name
  name                = "auth-api-main"
  database            = "auth"
  renderedValues      = data.template_file.auth-api-main.rendered
}

module "user-api-main" {
  source = "./modules/service"

  namespace           = kubernetes_namespace.main.metadata[0].name
  labels              = local.labels
  annotations         = merge(local.annotations, {
    "go-api-boilerplate.local/role"    = "api"
    "go-api-boilerplate.local/project" = "user-api-go"
  })
  imagePullSecretName = kubernetes_secret.regcred.metadata.0.name
  envSecretName       = kubernetes_secret.config.metadata.0.name
  name                = "user-api-main"
  database            = "user"
  renderedValues      = data.template_file.user-api-main.rendered
}
