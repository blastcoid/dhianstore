/**
 * Workload Identity Federation for GitHub Actions.
 *
 * Provisions a pool + OIDC provider + dedicated CI service account so workflows
 * in the allow-listed repos can authenticate to GCP without long-lived JSON
 * keys. Wire up in a workflow with google-github-actions/auth:
 *
 *   - uses: google-github-actions/auth@v2
 *     with:
 *       workload_identity_provider: <output: provider_name>
 *       service_account:            <output: service_account_email>
 *
 * Naming standard: <Unit>-<Env>-<Code>-<Feature> (e.g. dst-prd-wif-github).
 */

locals {
  # Workload Identity Federation Naming Standard
  wif_standard = {
    Unit    = var.unit
    Env     = var.env
    Code    = var.code
    Feature = var.feature
  }
  wif_naming_standard = "${local.wif_standard.Unit}-${local.wif_standard.Env}-${local.wif_standard.Code}-${local.wif_standard.Feature}"
}

module "workload_identity" {
  source = "../../modules/workload-identity"

  name                  = local.wif_naming_standard
  standard              = local.wif_standard
  project_id            = var.project_id
  repositories          = var.repositories
  attribute_condition   = var.attribute_condition
  service_account_roles = var.service_account_roles
}
