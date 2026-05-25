unit       = "dst"
env        = "prd"
code       = "wif"
feature    = "github"
project_id = "grey-playground"

attribute_condition = "attribute.repository_owner == 'blastcoid'"

repositories = [
  "blastcoid/dhianstore",
]

service_account_roles = [
  "roles/owner",
]
