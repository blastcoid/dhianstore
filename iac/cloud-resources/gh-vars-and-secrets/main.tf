/**
 * Publishes GitHub Actions variables and secrets to the dhianstore repository.
 *
 * Values live in:
 *   - terraform.tfvars     -> var.dhianstore_vars
 *   - secrets.auto.tfvars  -> var.dhianstore_secrets (gitignored)
 *
 * The locals retain a merge skeleton so values sourced from data sources or
 * remote_state can be added later without restructuring the module call.
 * Externally-sourced maps are empty today; add entries when needed.
 * External entries take precedence over caller-supplied ones on key collision.
 */

locals {
  # Placeholder for entries sourced from data sources or remote_state later.
  external_vars    = {}
  external_secrets = {}

  dhianstore_vars    = merge(var.dhianstore_vars, local.external_vars)
  dhianstore_secrets = merge(var.dhianstore_secrets, local.external_secrets)
}

module "github_vars_and_secrets" {
  source = "../../modules/gh-vars-and-secrets"

  repository = var.github_repository
  variables  = local.dhianstore_vars
  secrets    = local.dhianstore_secrets
}
