# change this value to trigger a change in Terraform and a new template to be generated
resource "null_resource" "template" {
  triggers = {
    value = "0"
  }
}

output "templates" {
  value = templatefile("${path.module}/templates.yml.tpl", {
    namespace   = local.cert-manager.namespace
    labels      = chomp(yamlencode(local.labels))
    annotations = chomp(yamlencode(merge(local.annotations, {
      "go-api-boilerplate.local/role"    = "web"
      "go-api-boilerplate.local/project" = "kubernetes"
    })))

    app     = local.app
    domains = local.domains
  })
}

data "template_file" "web-ui-main" {
  template = file("${path.module}/../cmd/web/main.yml")
}

data "template_file" "auth-api-main" {
  template = file("${path.module}/../cmd/auth/main.yml")
}

data "template_file" "user-api-main" {
  template = file("${path.module}/../cmd/user/main.yml")
}
