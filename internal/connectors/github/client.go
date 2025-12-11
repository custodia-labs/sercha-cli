package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	gh "github.com/google/go-github/v80/github"
	"golang.org/x/oauth2"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

const (
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// MaxRetries is the maximum number of retries for transient errors.
	MaxRetries = 3

	// RetryDelay is the initial delay between retries.
	RetryDelay = time.Second
)

// Client wraps the go-github client with helper methods.
type Client struct {
	gh            *gh.Client
	tokenProvider driven.TokenProvider
	rateLimiter   *RateLimiter
}

// NewClient creates a new GitHub API client with a token provider.
func NewClient(tokenProvider driven.TokenProvider) *Client {
	return &Client{
		tokenProvider: tokenProvider,
		rateLimiter:   NewRateLimiter(),
	}
}

// ensureClient initializes the go-github client if not already done.
// This is called lazily so we can get the token when needed.
func (c *Client) ensureClient(ctx context.Context) error {
	if c.gh != nil {
		return nil
	}

	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	tc.Timeout = DefaultTimeout
	c.gh = gh.NewClient(tc)

	return nil
}

// GitHub returns the underlying go-github client.
// Caller should call ensureClient first.
func (c *Client) GitHub() *gh.Client {
	return c.gh
}

// ListAllAccessibleRepos returns ALL repositories the authenticated user can access.
// This includes: owned repos, collaborator repos, and organization member repos.
func (c *Client) ListAllAccessibleRepos(ctx context.Context) ([]*gh.Repository, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	var allRepos []*gh.Repository

	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Visibility:  "all",                                    // public + private
		Affiliation: "owner,collaborator,organization_member", // all relationships
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return allRepos, ctx.Err()
		default:
		}

		// Wait for rate limit
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}

		repos, resp, err := c.gh.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, c.wrapError(err, "list repos")
		}

		// Update rate limiter from response
		c.updateRateLimitFromResponse(resp)

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// GetRepository fetches a single repository.
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	repository, resp, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, c.wrapError(err, "get repo")
	}

	c.updateRateLimitFromResponse(resp)
	return repository, nil
}

// GetTree fetches the entire tree for a repository recursively.
// This is efficient for getting all file paths in one API call.
func (c *Client) GetTree(ctx context.Context, owner, repo, sha string) (*gh.Tree, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	tree, resp, err := c.gh.Git.GetTree(ctx, owner, repo, sha, true) // recursive=true
	if err != nil {
		return nil, c.wrapError(err, "get tree")
	}

	c.updateRateLimitFromResponse(resp)
	return tree, nil
}

// GetBlob fetches a blob (file content) by its SHA.
func (c *Client) GetBlob(ctx context.Context, owner, repo, sha string) (*gh.Blob, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	blob, resp, err := c.gh.Git.GetBlob(ctx, owner, repo, sha)
	if err != nil {
		return nil, c.wrapError(err, "get blob")
	}

	c.updateRateLimitFromResponse(resp)
	return blob, nil
}

// GetFileContent fetches the content of a file.
// For files < 1MB, content is base64 encoded in the response.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	if err := c.ensureClient(ctx); err != nil {
		return "", err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait: %w", err)
	}

	opts := &gh.RepositoryContentGetOptions{Ref: ref}
	content, _, resp, err := c.gh.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", c.wrapError(err, "get contents")
	}

	c.updateRateLimitFromResponse(resp)

	if content == nil {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	decoded, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("decode content: %w", err)
	}
	return decoded, nil
}

// DownloadContents downloads a file larger than 1MB.
// Returns an io.ReadCloser that must be closed by the caller.
func (c *Client) DownloadContents(ctx context.Context, owner, repo, path, ref string) (io.ReadCloser, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	opts := &gh.RepositoryContentGetOptions{Ref: ref}
	rc, resp, err := c.gh.Repositories.DownloadContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, c.wrapError(err, "download contents")
	}

	c.updateRateLimitFromResponse(resp)
	return rc, nil
}

// ListIssues lists issues for a repository.
func (c *Client) ListIssues(
	ctx context.Context, owner, repo string, opts *gh.IssueListByRepoOptions,
) ([]*gh.Issue, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	var allIssues []*gh.Issue

	for {
		select {
		case <-ctx.Done():
			return allIssues, ctx.Err()
		default:
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}

		issues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, c.wrapError(err, "list issues")
		}

		c.updateRateLimitFromResponse(resp)
		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return allIssues, nil
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(
	ctx context.Context, owner, repo string, opts *gh.PullRequestListOptions,
) ([]*gh.PullRequest, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	var allPRs []*gh.PullRequest

	for {
		select {
		case <-ctx.Done():
			return allPRs, ctx.Err()
		default:
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}

		prs, resp, err := c.gh.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, c.wrapError(err, "list pull requests")
		}

		c.updateRateLimitFromResponse(resp)
		allPRs = append(allPRs, prs...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// RateLimit returns the current rate limit status.
func (c *Client) RateLimit(ctx context.Context) (*gh.RateLimits, error) {
	if err := c.ensureClient(ctx); err != nil {
		return nil, err
	}

	limits, _, err := c.gh.RateLimit.Get(ctx)
	if err != nil {
		return nil, c.wrapError(err, "get rate limit")
	}
	return limits, nil
}

// RateLimiter returns the rate limiter for external access.
func (c *Client) RateLimiter() *RateLimiter {
	return c.rateLimiter
}

// updateRateLimitFromResponse updates the rate limiter from GitHub response headers.
func (c *Client) updateRateLimitFromResponse(resp *gh.Response) {
	if resp == nil || resp.Response == nil {
		return
	}
	c.rateLimiter.UpdateFromResponse(resp.Response)
}

// wrapError converts go-github errors to our error types.
func (c *Client) wrapError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check for GitHub error response
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		return &APIError{
			StatusCode: ghErr.Response.StatusCode,
			Message:    ghErr.Message,
			URL:        ghErr.Response.Request.URL.String(),
		}
	}

	// Check for rate limit error
	var rateLimitErr *gh.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return &RateLimitError{
			ResetAt:   c.rateLimiter.ResetTime(),
			Remaining: c.rateLimiter.Remaining(),
			Limit:     c.rateLimiter.Limit(),
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// ValidateCredentials checks if the provided token is valid by making an API call.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait: %w", err)
	}

	_, resp, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return c.wrapError(err, "validate credentials")
	}

	c.updateRateLimitFromResponse(resp)
	return nil
}

// TokenProvider returns the token provider (used by other modules).
func (c *Client) TokenProvider() driven.TokenProvider {
	return c.tokenProvider
}

// NewClientWithHTTPClient creates a GitHub client with a custom http.Client.
// Useful for OAuth flows where the http.Client handles token refresh.
func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{
		gh:          gh.NewClient(httpClient),
		rateLimiter: NewRateLimiter(),
	}
}

// NewClientWithToken creates a GitHub client with a static access token.
// Works for both PAT and OAuth access tokens.
func NewClientWithToken(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	tc.Timeout = DefaultTimeout

	return &Client{
		gh:          gh.NewClient(tc),
		rateLimiter: NewRateLimiter(),
	}
}
