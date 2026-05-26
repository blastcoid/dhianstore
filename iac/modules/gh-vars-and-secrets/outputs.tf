output "variable_names" {
  description = "Names of the GitHub Actions variables managed by this module."
  value       = keys(var.variables)
}

output "secret_names" {
  description = "Names of the GitHub Actions secrets managed by this module. Values are not exposed."
  value       = nonsensitive(keys(var.secrets))
}
