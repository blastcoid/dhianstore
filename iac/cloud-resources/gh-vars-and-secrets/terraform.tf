terraform {
  required_version = ">= 1.14"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
  }

  backend "gcs" {
    bucket = "bls-tfstate"
    prefix = "dhianstore/prd/iac/github-vars-and-secrets"
  }
}
