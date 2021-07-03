terraform {
  backend "local" {
    path = "terraform.tfstate"
  }

  required_version = "~> 0.14"

  required_providers {
    kubernetes = {
      version = "~> 2.0.2"
      source  = "hashicorp/kubernetes"
    }
    helm = {
      version = "~> 2.1.0"
      source  = "hashicorp/helm"
    }
  }
}
