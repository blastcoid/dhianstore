# GCP Settings
variable "name" {
  type        = string
  description = "Workload Identity Pool ID (also used as SA account_id). Must follow naming standard: <unit>-<env>-<code>-<feature>."

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,30}[a-z0-9]$", var.name))
    error_message = "name must be 4-32 chars, lowercase alphanumerics or hyphens, start with a letter, end with a letter or digit."
  }
}

variable "standard" {
  type        = map(string)
  description = "The standard naming convention for resources."
}

# Workload Identity Federation arguments
variable "project_id" {
  type        = string
  description = "GCP project ID hosting the WIF pool and CI service account."
}

variable "provider_id" {
  type        = string
  description = "OIDC provider ID created inside the pool. Defaults to 'github'."
  default     = "github"
}

variable "oidc_issuer_uri" {
  type        = string
  description = "OIDC issuer URI. Defaults to GitHub Actions token issuer."
  default     = "https://token.actions.githubusercontent.com"
}

variable "attribute_mapping" {
  type        = map(string)
  description = "OIDC claim to attribute mapping. Defaults match GitHub Actions claims."
  default = {
    "google.subject"             = "assertion.sub"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
    "attribute.ref"              = "assertion.ref"
    "attribute.workflow"         = "assertion.workflow"
    "attribute.actor"            = "assertion.actor"
  }
}

variable "attribute_condition" {
  type        = string
  description = "CEL expression restricting which OIDC tokens may exchange for a Google token. Must reference mapped attributes from attribute_mapping (e.g. 'attribute.repository_owner == \"my-org\"'), not raw 'assertion.*' claims. See https://cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines#conditions"
  default     = null
}

variable "repositories" {
  type        = set(string)
  description = "GitHub repos ('<owner>/<repo>') whose workflows may impersonate the CI service account."
  default     = []
}

variable "service_account_roles" {
  type        = set(string)
  description = "Project-level IAM roles granted to the CI service account in project_id."
  default     = []
}
