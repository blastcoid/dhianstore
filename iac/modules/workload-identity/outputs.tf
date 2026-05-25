output "pool_name" {
  description = "Fully-qualified Workload Identity Pool resource name."
  value       = google_iam_workload_identity_pool.this.name
}

output "provider_name" {
  description = "Fully-qualified OIDC provider resource name. Use this as 'workload_identity_provider' in google-github-actions/auth."
  value       = google_iam_workload_identity_pool_provider.this.name
}

output "service_account_email" {
  description = "Email of the CI service account that workflows impersonate. Use as 'service_account' in google-github-actions/auth."
  value       = google_service_account.ci.email
}

output "service_account_member" {
  description = "IAM member string for the CI SA, usable directly in google_*_iam_member resources elsewhere."
  value       = "serviceAccount:${google_service_account.ci.email}"
}
