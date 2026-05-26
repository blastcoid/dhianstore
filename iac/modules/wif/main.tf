/**
 * Workload Identity Federation pool + OIDC provider + dedicated CI service
 * account that external workflows can impersonate without long-lived keys.
 *
 * Defaults are tuned for GitHub Actions (issuer URI, attribute mapping).
 * Override provider_id, oidc_issuer_uri, and attribute_mapping to reuse this
 * module for GitLab, Bitbucket, or any other OIDC source.
 *
 * Note: deleted WIF pools enter a 30-day soft-deleted state during which the
 * pool ID is reserved. Recreating a pool with the same name within that window
 * will fail.
 */

data "google_project" "this" {
  project_id = var.project_id
}

resource "google_iam_workload_identity_pool" "this" {
  project                   = var.project_id
  workload_identity_pool_id = var.name
  display_name              = var.name
  description               = "WIF pool for ${var.standard.Unit}/${var.standard.Env}/${var.standard.Code}/${var.standard.Feature}"
}

resource "google_iam_workload_identity_pool_provider" "this" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.this.workload_identity_pool_id
  workload_identity_pool_provider_id = var.provider_id
  display_name                       = var.provider_id

  attribute_mapping   = var.attribute_mapping
  attribute_condition = var.attribute_condition

  oidc {
    issuer_uri = var.oidc_issuer_uri
  }
}

resource "google_service_account" "ci" {
  project      = var.project_id
  account_id   = var.name
  display_name = "CI runner impersonated via WIF (${var.name})"
}

# Per-repo workloadIdentityUser binding on the CI SA. Each binding restricts
# impersonation to a specific GitHub repository.
resource "google_service_account_iam_member" "workload_identity_user" {
  for_each = var.repositories

  service_account_id = google_service_account.ci.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.this.number}/locations/global/workloadIdentityPools/${google_iam_workload_identity_pool.this.workload_identity_pool_id}/attribute.repository/${each.key}"
}

resource "google_project_iam_member" "ci_roles" {
  for_each = var.service_account_roles

  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.ci.email}"
}
