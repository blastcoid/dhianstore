<!-- BEGIN_TF_DOCS -->
# Workload Identity Federation Module
Workload Identity Federation pool + OIDC provider + dedicated CI service
account that external workflows can impersonate without long-lived keys.

Defaults are tuned for GitHub Actions (issuer URI, attribute mapping).
Override provider\_id, oidc\_issuer\_uri, and attribute\_mapping to reuse this
module for GitLab, Bitbucket, or any other OIDC source.

Note: deleted WIF pools enter a 30-day soft-deleted state during which the
pool ID is reserved. Recreating a pool with the same name within that window
will fail.

## Providers

| Name | Version |
| ---- | ------- |
| google | n/a |

## Resources

| Name | Type |
| ---- | ---- |
| [google_iam_workload_identity_pool.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/iam_workload_identity_pool) | resource |
| [google_iam_workload_identity_pool_provider.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/iam_workload_identity_pool_provider) | resource |
| [google_project_iam_member.ci_roles](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_service_account.ci](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_member.workload_identity_user](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_member) | resource |
| [google_project.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| attribute\_condition | CEL expression restricting which OIDC tokens may exchange for a Google token. Must reference mapped attributes from attribute\_mapping (e.g. 'attribute.repository\_owner == "my-org"'), not raw 'assertion.*' claims. See https://cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines#conditions | `string` | `null` | no |
| attribute\_mapping | OIDC claim to attribute mapping. Defaults match GitHub Actions claims. | `map(string)` | ```{ "attribute.actor": "assertion.actor", "attribute.ref": "assertion.ref", "attribute.repository": "assertion.repository", "attribute.repository_owner": "assertion.repository_owner", "attribute.workflow": "assertion.workflow", "google.subject": "assertion.sub" }``` | no |
| name | Workload Identity Pool ID (also used as SA account\_id). Must follow naming standard: <unit>-<env>-<code>-<feature>. | `string` | n/a | yes |
| oidc\_issuer\_uri | OIDC issuer URI. Defaults to GitHub Actions token issuer. | `string` | `"https://token.actions.githubusercontent.com"` | no |
| project\_id | GCP project ID hosting the WIF pool and CI service account. | `string` | n/a | yes |
| provider\_id | OIDC provider ID created inside the pool. Defaults to 'github'. | `string` | `"github"` | no |
| repositories | GitHub repos ('<owner>/<repo>') whose workflows may impersonate the CI service account. | `set(string)` | `[]` | no |
| service\_account\_roles | Project-level IAM roles granted to the CI service account in project\_id. | `set(string)` | `[]` | no |
| standard | The standard naming convention for resources. | `map(string)` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| pool\_name | Fully-qualified Workload Identity Pool resource name. |
| project\_id | GCP project ID hosting the WIF pool and CI service account. |
| provider\_name | Fully-qualified OIDC provider resource name. Use this as 'workload\_identity\_provider' in google-github-actions/auth. |
| service\_account\_email | Email of the CI service account that workflows impersonate. Use as 'service\_account' in google-github-actions/auth. |
| service\_account\_member | IAM member string for the CI SA, usable directly in google\_*\_iam\_member resources elsewhere. |
<!-- END_TF_DOCS -->