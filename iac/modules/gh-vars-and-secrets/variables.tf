# GitHub target
variable "repository" {
  type        = string
  description = "GitHub repository name (without owner prefix). Provider config in the caller scopes API calls to the owner."
}

# Payload
variable "variables" {
  type        = map(string)
  description = "Map of plain-text GitHub Actions variables (NAME => value)."
  default     = {}
}

variable "secrets" {
  type        = map(string)
  description = "Map of GitHub Actions secrets (NAME => plaintext value). Stored sensitive in state."
  default     = {}
  sensitive   = true
}
