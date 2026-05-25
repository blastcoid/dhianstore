# Naming Standard
variable "unit" {
  type        = string
  description = "Business unit code."
}

variable "env" {
  type        = string
  description = "Stage environment where the infrastructure will be deployed."

  validation {
    condition     = contains(["dev", "stg", "prd"], var.env)
    error_message = "env must be one of dev, stg, prd."
  }
}

variable "code" {
  type        = string
  description = "Resource-category code, used as the 'Code' component of the naming standard (e.g. 'wif' for Workload Identity Federation)."
}

variable "feature" {
  type        = string
  description = "Feature slug, used as the 'Feature' component of the naming standard (e.g. 'github' for GitHub Actions)."
}

# Project
variable "project_id" {
  type        = string
  description = "GCP project ID that hosts the WIF pool and the CI service account."
}

# Workload Identity Federation arguments
variable "repositories" {
  type        = set(string)
  description = "GitHub repos ('<owner>/<repo>') whose Actions workflows may impersonate the CI service account."
}

variable "attribute_condition" {
  type        = string
  description = "CEL expression restricting which OIDC tokens are accepted. Must reference attribute_mapping outputs (e.g. attribute.repository_owner == '<org>'), not raw assertion.* claims. Recommended for GitHub: pin to your org."
  default     = null
}

variable "service_account_roles" {
  type        = set(string)
  description = "Project-level IAM roles granted to the CI service account."
  default     = []
}
