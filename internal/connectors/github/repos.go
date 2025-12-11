package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

// ListAllRepos returns all repositories accessible to the authenticated user.
// This is the primary method for indexing - it gets ALL repos the user can access:
// owned repositories, collaborator repositories, and organization member repositories.
func ListAllRepos(ctx context.Context, client *Client) ([]*gh.Repository, error) {
	return client.ListAllAccessibleRepos(ctx)
}

// FilterRepos filters repositories based on criteria.
func FilterRepos(repos []*gh.Repository, includeArchived, includeForks bool) []*gh.Repository {
	filtered := make([]*gh.Repository, 0, len(repos))
	for _, r := range repos {
		if r.GetArchived() && !includeArchived {
			continue
		}
		if r.GetFork() && !includeForks {
			continue
		}
		if r.GetDisabled() {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// GetTree retrieves the full tree for a repository at a given ref.
// Uses recursive=1 to get all files in one call.
func GetTree(ctx context.Context, client *Client, owner, repo, ref string) (*gh.Tree, error) {
	return client.GetTree(ctx, owner, repo, ref)
}
