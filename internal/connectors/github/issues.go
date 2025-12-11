package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gh "github.com/google/go-github/v80/github"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MIMETypeGitHubIssue is the custom MIME type for GitHub issues.
const MIMETypeGitHubIssue = "application/vnd.github.issue+json"

// IssueContent is the JSON structure for the issue RawDocument content.
type IssueContent struct {
	Number    int              `json:"number"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	State     string           `json:"state"`
	Author    string           `json:"author"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Labels    []string         `json:"labels"`
	Assignees []string         `json:"assignees"`
	Milestone string           `json:"milestone,omitempty"`
	Comments  []CommentContent `json:"comments"`
}

// CommentContent represents a comment in the issue content.
type CommentContent struct {
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// FetchIssues retrieves all issues (excluding PRs) from a repository.
func FetchIssues(
	ctx context.Context, client *Client, repo *gh.Repository, since time.Time,
) ([]domain.RawDocument, time.Time, error) {
	if !repo.GetHasIssues() {
		return nil, since, nil
	}

	owner := repo.GetOwner().GetLogin()
	name := repo.GetName()

	docs := make([]domain.RawDocument, 0)
	var latestUpdate time.Time

	// Build query options
	opts := &gh.IssueListByRepoOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "asc",
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}
	if !since.IsZero() {
		opts.Since = since
	}

	issues, err := client.ListIssues(ctx, owner, name, opts)
	if err != nil {
		return nil, since, fmt.Errorf("list issues: %w", err)
	}

	for _, issue := range issues {
		// Skip pull requests (they show up in issues endpoint too).
		if issue.IsPullRequest() {
			continue
		}

		// Track latest update.
		if issue.GetUpdatedAt().Time.After(latestUpdate) {
			latestUpdate = issue.GetUpdatedAt().Time
		}

		// Fetch comments.
		comments, commErr := FetchIssueComments(ctx, client, owner, name, issue.GetNumber())
		if commErr != nil {
			comments = nil
		}

		// Build content.
		content := buildIssueContent(issue, comments)
		contentJSON, jsonErr := json.Marshal(content)
		if jsonErr != nil {
			continue
		}

		// Build labels slice
		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.GetName()
		}

		// Build assignees slice
		assignees := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			assignees[i] = a.GetLogin()
		}

		doc := domain.RawDocument{
			SourceID: "", // Will be set by connector
			URI:      buildIssueURI(owner, name, issue.GetNumber()),
			MIMEType: MIMETypeGitHubIssue,
			Content:  contentJSON,
			Metadata: map[string]any{
				"type":       "issue",
				"owner":      owner,
				"repo":       name,
				"number":     issue.GetNumber(),
				"title":      issue.GetTitle(),
				"state":      issue.GetState(),
				"author":     issue.GetUser().GetLogin(),
				"labels":     labels,
				"assignees":  assignees,
				"comments":   issue.GetComments(),
				"html_url":   issue.GetHTMLURL(),
				"created_at": issue.GetCreatedAt().Format(time.RFC3339),
				"updated_at": issue.GetUpdatedAt().Format(time.RFC3339),
			},
		}
		docs = append(docs, doc)
	}

	return docs, latestUpdate, nil
}

// FetchIssueComments retrieves all comments for an issue.
func FetchIssueComments(
	ctx context.Context, client *Client, owner, repo string, issueNumber int,
) ([]*gh.IssueComment, error) {
	if err := client.ensureClient(ctx); err != nil {
		return nil, err
	}

	var allComments []*gh.IssueComment

	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		select {
		case <-ctx.Done():
			return allComments, ctx.Err()
		default:
		}

		if err := client.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}

		comments, resp, err := client.gh.Issues.ListComments(ctx, owner, repo, issueNumber, opts)
		if err != nil {
			return nil, client.wrapError(err, "list comments")
		}

		client.updateRateLimitFromResponse(resp)
		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// buildIssueContent creates the IssueContent structure.
func buildIssueContent(issue *gh.Issue, comments []*gh.IssueComment) IssueContent {
	labels := make([]string, len(issue.Labels))
	for i, l := range issue.Labels {
		labels[i] = l.GetName()
	}

	assignees := make([]string, len(issue.Assignees))
	for i, a := range issue.Assignees {
		assignees[i] = a.GetLogin()
	}

	var milestone string
	if issue.Milestone != nil {
		milestone = issue.Milestone.GetTitle()
	}

	commentContents := make([]CommentContent, len(comments))
	for i, c := range comments {
		commentContents[i] = CommentContent{
			Author:    c.GetUser().GetLogin(),
			Body:      c.GetBody(),
			CreatedAt: c.GetCreatedAt().Time,
		}
	}

	return IssueContent{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		State:     issue.GetState(),
		Author:    issue.GetUser().GetLogin(),
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
		Labels:    labels,
		Assignees: assignees,
		Milestone: milestone,
		Comments:  commentContents,
	}
}

// buildIssueURI creates a URI for an issue.
func buildIssueURI(owner, repo string, number int) string {
	return fmt.Sprintf("github://%s/%s/issues/%d", owner, repo, number)
}
