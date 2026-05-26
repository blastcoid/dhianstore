/**
 * # GitHub Actions Variables and Secrets Module
 * Publishes GitHub Actions variables and secrets to a single repository.
 *
 * Caller is responsible for:
 *   - Configuring the github provider (owner, token).
 *   - Building/merging the variables and secrets maps (e.g. combining
 *     tfvars-supplied entries with tfstate-sourced ones from other stacks).
 *
 * To extend: add a new entry to the variables or secrets map at the caller.
 */

resource "github_actions_variable" "this" {
  for_each = var.variables

  repository    = var.repository
  variable_name = each.key
  value         = each.value
}

resource "github_actions_secret" "this" {
  # `var.secrets` is a sensitive map; for_each must iterate over a non-sensitive
  # value because resource instance keys appear in plan output. Keys themselves
  # are not secret (only the values), so we iterate keys and look up values
  # inside the block — Terraform tracks the value as sensitive.
  for_each = nonsensitive(toset(keys(var.secrets)))

  repository      = var.repository
  secret_name     = each.key
  plaintext_value = var.secrets[each.key]
}
