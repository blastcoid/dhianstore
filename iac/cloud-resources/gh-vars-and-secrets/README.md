<!-- BEGIN_TF_DOCS -->
Publishes GitHub Actions variables and secrets to the dhianstore repository.

Values live in:
  - terraform.tfvars     -> var.dhianstore\_vars
  - secrets.auto.tfvars  -> var.dhianstore\_secrets (gitignored)

The locals retain a merge skeleton so values sourced from data sources or
remote\_state can be added later without restructuring the module call.
Externally-sourced maps are empty today; add entries when needed.
External entries take precedence over caller-supplied ones on key collision.

## Requirements

| Name | Version |
| ---- | ------- |
| terraform | >= 1.14 |
| github | ~> 6.0 |

## Modules

| Name | Source | Version |
| ---- | ------ | ------- |
| github\_vars\_and\_secrets | ../../modules/gh-vars-and-secrets | n/a |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| dhianstore\_secrets | GitHub Actions secrets to publish to the dhianstore repo. Defined in secrets.auto.tfvars (gitignored). Merged with externally-sourced entries; external values win on collision. | `map(string)` | `{}` | no |
| dhianstore\_vars | Plain-text GitHub Actions variables to publish to the dhianstore repo. Defined in terraform.tfvars. Merged with externally-sourced entries; external values win on collision. | `map(string)` | `{}` | no |
| github\_owner | GitHub organization or user that owns the target repository. | `string` | n/a | yes |
| github\_repository | GitHub repository name (without owner prefix) that receives the variables and secrets. | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| published\_secrets | Names of the GitHub Actions secrets managed by this stack. Values are not exposed. |
| published\_variables | Names of the GitHub Actions variables managed by this stack. |
<!-- END_TF_DOCS -->