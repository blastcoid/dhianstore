/**
 * GitHub provider authenticates via the GITHUB_TOKEN env var (set in the
 * operator's ~/.zprofile). The owner scopes API calls to the target org.
 */

provider "github" {
  owner = var.github_owner
}
