variable "kube_system_namespace" {
  type = string
  default = "kube-system"
}

variable "kube_config" {
  type = string
  default = "~/.kube/config"
}

variable "config_context" {
  type = string
  default = "go-api-boilerplate"
}

locals {
  app = "go-api-boilerplate"
  env = "prod"

  labels = {
    "go-api-boilerplate.local/name"       = local.app
    "go-api-boilerplate.local/env"        = local.env
    "go-api-boilerplate.local/managed-by" = "helm"
    "go-api-boilerplate.local/owner"      = "vardius"
  }

  tcp = {
    27017: "go-api-boilerplate/mongodb:27017"
  }

  annotations = {
    deployed-by = "vardius@gmail.com"
    repo        = "https://github.com/vardius/go-api-boilerplate"
  }

  mongodb = {
    version   = "v10.7.0"
    namespace = kubernetes_namespace.main.metadata[0].name
  }

  ingress-nginx = {
    version   = "v3.30.0"
    namespace = kubernetes_namespace.main.metadata[0].name
  }

  cert-manager = {
    version   = "v1.3.1" # READ THIS FIRST VERSION BY VERSION! https://cert-manager.io/docs/installation/upgrading/
    namespace = kubernetes_namespace.main.metadata[0].name
  }

  domains = [
    {
      name  = "go-api-boilerplate.local",
      paths = [
        {
          path        = "/(|$)(.*)"
          serviceName = module.web-ui-main.name
          servicePort = 3000
        }
      ],
      subdomains = [
        {
          name  = "api"
          paths = [
            {
              path        = "/auth(/|$)(.*)"
              serviceName = module.auth-api-main.name
              servicePort = 3000
            },
            {
              path        = "/users(/|$)(.*)"
              serviceName = module.user-api-main.name
              servicePort = 3000
            },
          ]
        },
        {
          name  = "maildev"
          paths = [
            {
              path        = "/(|$)(.*)"
              serviceName = helm_release.maildev.metadata[0].name
              servicePort = 1080
            }
          ]
        }
      ]
    }
  ]
}
