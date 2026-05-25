terraform {
  required_version = ">= 1.14"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }

  backend "gcs" {
    bucket = "bls-tfstate"
    prefix = "dhianstore/prd/iac/workload-identity-github"
  }
}
