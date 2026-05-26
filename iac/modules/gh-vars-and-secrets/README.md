<!-- BEGIN_TF_DOCS -->
Publishes GitHub Actions variables and secrets to a single repository.

Caller is responsible for:
  - Configuring the github provider (owner, token).
  - Building/merging the variables and secrets maps (e.g. combining
    tfvars-supplied entries with tfstate-sourced ones from other stacks).

To extend: add a new entry to the variables or secrets map at the caller.

## Requirements

| Name | Version |
| ---- | ------- |
| terraform | >= 1.14 |
| github | ~> 6.0 |

## Providers

| Name | Version |
| ---- | ------- |
| github | ~> 6.0 |

## Resources

| Name | Type |
| ---- | ---- |
| [github_actions_secret.this](https://registry.terraform.io/providers/integrations/github/latest/docs/resources/actions_secret) | resource |
| [github_actions_variable.this](https://registry.terraform.io/providers/integrations/github/latest/docs/resources/actions_variable) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| repository | GitHub repository name (without owner prefix). Provider config in the caller scopes API calls to the owner. | `string` | n/a | yes |
| secrets | Map of GitHub Actions secrets (NAME => plaintext value). Stored sensitive in state. | `map(string)` | `{}` | no |
| variables | Map of plain-text GitHub Actions variables (NAME => value). | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| secret\_names | Names of the GitHub Actions secrets managed by this module. Values are not exposed. |
| variable\_names | Names of the GitHub Actions variables managed by this module. |
<!-- END_TF_DOCS -->