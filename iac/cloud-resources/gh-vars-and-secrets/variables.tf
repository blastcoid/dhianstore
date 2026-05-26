# GitHub target
variable "github_owner" {
  type        = string
  description = "GitHub organization or user that owns the target repository."
}

variable "github_repository" {
  type        = string
  description = "GitHub repository name (without owner prefix) that receives the variables and secrets."
}

# Caller-supplied vars/secrets — merged with externally-sourced entries in locals.
# Keys present in the external_* locals take precedence over caller-supplied
# keys with the same name (external is authoritative when both are populated).
variable "dhianstore_vars" {
  type        = map(string)
  description = "Plain-text GitHub Actions variables to publish to the dhianstore repo. Defined in terraform.tfvars. Merged with externally-sourced entries; external values win on collision."
  default     = {}
}

variable "dhianstore_secrets" {
  type        = map(string)
  description = "GitHub Actions secrets to publish to the dhianstore repo. Defined in secrets.auto.tfvars (gitignored). Merged with externally-sourced entries; external values win on collision."
  default     = {}
  sensitive   = true
}
