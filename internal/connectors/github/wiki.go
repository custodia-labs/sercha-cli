package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v80/github"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// FetchWikiPages retrieves wiki pages from a repository.
// Note: GitHub's REST API has limited wiki support. Wiki pages are accessed
// via the repo's wiki git repository at {repo}.wiki.git.
// For simplicity, we fetch the wiki page list and content via API where available.
func FetchWikiPages(ctx context.Context, client *Client, repo *gh.Repository) ([]domain.RawDocument, string, error) {
	if !repo.GetHasWiki() {
		return nil, "", ErrWikiDisabled
	}

	owner := repo.GetOwner().GetLogin()
	name := repo.GetName()

	// GitHub doesn't have a direct API for wiki pages
	// The wiki is a separate git repository: https://github.com/{owner}/{repo}.wiki.git
	// We can try to access it via the repos API but it's limited

	// First, check if wiki has any pages by trying to get the wiki git tree
	// Wiki repositories are accessed as {owner}/{repo}.wiki
	wikiRepoName := name + ".wiki"

	// Try to get the tree for the wiki repository
	tree, err := client.GetTree(ctx, owner, wikiRepoName, "master")
	if err != nil {
		// Wiki might not exist or be empty
		if IsNotFound(err) || IsForbidden(err) {
			return nil, "", ErrWikiDisabled
		}
		return nil, "", err
	}

	docs := make([]domain.RawDocument, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}

		path := entry.GetPath()

		// Only process .md files
		if len(path) < 3 || path[len(path)-3:] != ".md" {
			continue
		}

		// Fetch content
		content, err := fetchWikiBlobContent(ctx, client, owner, wikiRepoName, entry.GetSHA())
		if err != nil {
			continue
		}

		// Extract title from filename (remove .md extension)
		title := path
		if len(title) > 3 {
			title = title[:len(title)-3]
		}

		doc := domain.RawDocument{
			SourceID: "", // Will be set by connector
			URI:      buildWikiURI(owner, name, title),
			MIMEType: "text/markdown", // Wiki pages are markdown
			Content:  content,
			Metadata: map[string]any{
				"type":     "wiki",
				"owner":    owner,
				"repo":     name,
				"title":    title,
				"sha":      entry.GetSHA(),
				"html_url": fmt.Sprintf("https://github.com/%s/%s/wiki/%s", owner, name, title),
			},
		}
		docs = append(docs, doc)
	}

	return docs, tree.GetSHA(), nil
}

// fetchWikiBlobContent fetches the content of a wiki blob and decodes it.
func fetchWikiBlobContent(ctx context.Context, client *Client, owner, repo, sha string) ([]byte, error) {
	blob, err := client.GetBlob(ctx, owner, repo, sha)
	if err != nil {
		return nil, err
	}

	// Decode base64 content
	if blob.GetEncoding() == "base64" {
		// Remove any whitespace from base64 content
		content := strings.ReplaceAll(blob.GetContent(), "\n", "")
		content = strings.ReplaceAll(content, "\r", "")
		return base64.StdEncoding.DecodeString(content)
	}

	return []byte(blob.GetContent()), nil
}

// buildWikiURI creates a URI for a wiki page.
func buildWikiURI(owner, repo, title string) string {
	return fmt.Sprintf("github://%s/%s/wiki/%s", owner, repo, title)
}
