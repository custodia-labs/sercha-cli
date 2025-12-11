package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gh "github.com/google/go-github/v80/github"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MIMETypeGitHubPull is the custom MIME type for GitHub pull requests.
const MIMETypeGitHubPull = "application/vnd.github.pull+json"

// PRContent is the JSON structure for the PR RawDocument content.
type PRContent struct {
	Number       int              `json:"number"`
	Title        string           `json:"title"`
	Body         string           `json:"body"`
	State        string           `json:"state"`
	Draft        bool             `json:"draft"`
	Merged       bool             `json:"merged"`
	Author       string           `json:"author"`
	HeadBranch   string           `json:"head_branch"`
	BaseBranch   string           `json:"base_branch"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Labels       []string         `json:"labels"`
	Assignees    []string         `json:"assignees"`
	Reviewers    []string         `json:"reviewers"`
	Additions    int              `json:"additions"`
	Deletions    int              `json:"deletions"`
	ChangedFiles int              `json:"changed_files"`
	Comments     []CommentContent `json:"comments"`
	Reviews      []ReviewContent  `json:"reviews"`
}

// ReviewContent represents a review in the PR content.
type ReviewContent struct {
	Author      string    `json:"author"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// FetchPullRequests retrieves all pull requests from a repository.
func FetchPullRequests(
	ctx context.Context, client *Client, repo *gh.Repository, since time.Time,
) ([]domain.RawDocument, time.Time, error) {
	owner := repo.GetOwner().GetLogin()
	name := repo.GetName()

	docs := make([]domain.RawDocument, 0)
	var latestUpdate time.Time

	// Build query options
	opts := &gh.PullRequestListOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "asc",
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	prs, err := client.ListPullRequests(ctx, owner, name, opts)
	if err != nil {
		return nil, since, fmt.Errorf("list pull requests: %w", err)
	}

	for _, pr := range prs {
		// Skip if before since timestamp.
		if !since.IsZero() && pr.GetUpdatedAt().Time.Before(since) {
			continue
		}

		// Track latest update.
		if pr.GetUpdatedAt().Time.After(latestUpdate) {
			latestUpdate = pr.GetUpdatedAt().Time
		}

		// Fetch comments and reviews. Errors are non-fatal.
		comments, commErr := FetchIssueComments(ctx, client, owner, name, pr.GetNumber())
		if commErr != nil {
			comments = nil
		}
		reviews, revErr := FetchPRReviews(ctx, client, owner, name, pr.GetNumber())
		if revErr != nil {
			reviews = nil
		}

		// Build content.
		content := buildPRContent(pr, comments, reviews)
		contentJSON, jsonErr := json.Marshal(content)
		if jsonErr != nil {
			continue
		}

		doc := buildPRDocument(owner, name, pr, contentJSON)
		docs = append(docs, doc)
	}

	return docs, latestUpdate, nil
}

// buildPRDocument creates a RawDocument from a pull request.
func buildPRDocument(owner, name string, pr *gh.PullRequest, contentJSON []byte) domain.RawDocument {
	// Build labels slice.
	labels := make([]string, len(pr.Labels))
	for i, l := range pr.Labels {
		labels[i] = l.GetName()
	}

	// Build assignees slice.
	assignees := make([]string, len(pr.Assignees))
	for i, a := range pr.Assignees {
		assignees[i] = a.GetLogin()
	}

	// Build reviewers slice.
	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, r := range pr.RequestedReviewers {
		reviewers[i] = r.GetLogin()
	}

	// Determine effective state.
	state := pr.GetState()
	if pr.GetMerged() {
		state = "merged"
	}

	return domain.RawDocument{
		SourceID: "", // Will be set by connector.
		URI:      buildPRURI(owner, name, pr.GetNumber()),
		MIMEType: MIMETypeGitHubPull,
		Content:  contentJSON,
		Metadata: map[string]any{
			"type":          "pull_request",
			"owner":         owner,
			"repo":          name,
			"number":        pr.GetNumber(),
			"title":         pr.GetTitle(),
			"state":         state,
			"draft":         pr.GetDraft(),
			"merged":        pr.GetMerged(),
			"author":        pr.GetUser().GetLogin(),
			"head_branch":   pr.GetHead().GetRef(),
			"base_branch":   pr.GetBase().GetRef(),
			"labels":        labels,
			"assignees":     assignees,
			"reviewers":     reviewers,
			"additions":     pr.GetAdditions(),
			"deletions":     pr.GetDeletions(),
			"changed_files": pr.GetChangedFiles(),
			"html_url":      pr.GetHTMLURL(),
			"created_at":    pr.GetCreatedAt().Format(time.RFC3339),
			"updated_at":    pr.GetUpdatedAt().Format(time.RFC3339),
		},
	}
}

// FetchPRReviews retrieves all reviews for a pull request.
func FetchPRReviews(
	ctx context.Context, client *Client, owner, repo string, prNumber int,
) ([]*gh.PullRequestReview, error) {
	if err := client.ensureClient(ctx); err != nil {
		return nil, err
	}

	var allReviews []*gh.PullRequestReview

	opts := &gh.ListOptions{PerPage: 100}

	for {
		select {
		case <-ctx.Done():
			return allReviews, ctx.Err()
		default:
		}

		if err := client.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}

		reviews, resp, err := client.gh.PullRequests.ListReviews(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, client.wrapError(err, "list reviews")
		}

		client.updateRateLimitFromResponse(resp)
		allReviews = append(allReviews, reviews...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReviews, nil
}

// buildPRContent creates the PRContent structure.
func buildPRContent(pr *gh.PullRequest, comments []*gh.IssueComment, reviews []*gh.PullRequestReview) PRContent {
	labels := make([]string, len(pr.Labels))
	for i, l := range pr.Labels {
		labels[i] = l.GetName()
	}

	assignees := make([]string, len(pr.Assignees))
	for i, a := range pr.Assignees {
		assignees[i] = a.GetLogin()
	}

	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, r := range pr.RequestedReviewers {
		reviewers[i] = r.GetLogin()
	}

	commentContents := make([]CommentContent, len(comments))
	for i, c := range comments {
		commentContents[i] = CommentContent{
			Author:    c.GetUser().GetLogin(),
			Body:      c.GetBody(),
			CreatedAt: c.GetCreatedAt().Time,
		}
	}

	reviewContents := make([]ReviewContent, len(reviews))
	for i, r := range reviews {
		reviewContents[i] = ReviewContent{
			Author:      r.GetUser().GetLogin(),
			State:       r.GetState(),
			Body:        r.GetBody(),
			SubmittedAt: r.GetSubmittedAt().Time,
		}
	}

	return PRContent{
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		Body:         pr.GetBody(),
		State:        pr.GetState(),
		Draft:        pr.GetDraft(),
		Merged:       pr.GetMerged(),
		Author:       pr.GetUser().GetLogin(),
		HeadBranch:   pr.GetHead().GetRef(),
		BaseBranch:   pr.GetBase().GetRef(),
		CreatedAt:    pr.GetCreatedAt().Time,
		UpdatedAt:    pr.GetUpdatedAt().Time,
		Labels:       labels,
		Assignees:    assignees,
		Reviewers:    reviewers,
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		Comments:     commentContents,
		Reviews:      reviewContents,
	}
}

// buildPRURI creates a URI for a pull request.
func buildPRURI(owner, repo string, number int) string {
	return fmt.Sprintf("github://%s/%s/pull/%d", owner, repo, number)
}
