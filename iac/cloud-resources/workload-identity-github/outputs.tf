output "provider_name" {
  description = "Fully-qualified WIF provider name. Paste into 'workload_identity_provider' in google-github-actions/auth."
  value       = module.workload_identity.provider_name
}

output "service_account_email" {
  description = "Email of the CI SA. Paste into 'service_account' in google-github-actions/auth."
  value       = module.workload_identity.service_account_email
}
