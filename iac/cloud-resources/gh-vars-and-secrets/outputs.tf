output "published_variables" {
  description = "Names of the GitHub Actions variables managed by this stack."
  value       = module.github_vars_and_secrets.variable_names
}

output "published_secrets" {
  description = "Names of the GitHub Actions secrets managed by this stack. Values are not exposed."
  value       = module.github_vars_and_secrets.secret_names
}
