package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Connector implements the interface.
var _ driven.Connector = (*Connector)(nil)

// Connector fetches documents from GitHub repositories.
type Connector struct {
	sourceID      string
	config        *Config
	client        *Client
	tokenProvider driven.TokenProvider
	mu            sync.Mutex
	closed        bool
}

// New creates a new GitHub connector.
func New(sourceID string, cfg *Config, tokenProvider driven.TokenProvider) *Connector {
	return &Connector{
		sourceID:      sourceID,
		config:        cfg,
		tokenProvider: tokenProvider,
		client:        NewClient(tokenProvider),
	}
}

// Type returns the connector type identifier.
func (c *Connector) Type() string {
	return "github"
}

// SourceID returns the source identifier.
func (c *Connector) SourceID() string {
	return c.sourceID
}

// Capabilities returns the connector's capabilities.
func (c *Connector) Capabilities() driven.ConnectorCapabilities {
	return driven.ConnectorCapabilities{
		SupportsIncremental:  true,
		SupportsWatch:        false, // No webhooks in CLI
		SupportsHierarchy:    true,  // Files have directories
		SupportsBinary:       false, // Text only
		RequiresAuth:         true,
		SupportsValidation:   true,
		SupportsCursorReturn: true,
		SupportsPartialSync:  true, // Can resume
		SupportsRateLimiting: true,
		SupportsPagination:   true,
	}
}

// Validate checks if the GitHub connector is properly configured.
func (c *Connector) Validate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return domain.ErrConnectorClosed
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate credentials by making an API call
	if err := c.client.ValidateCredentials(ctx); err != nil {
		if IsUnauthorized(err) {
			return domain.ErrAuthInvalid
		}
		return fmt.Errorf("%w: %w", domain.ErrAuthRequired, err)
	}

	return nil
}

// FullSync fetches all documents from GitHub.
func (c *Connector) FullSync(ctx context.Context) (<-chan domain.RawDocument, <-chan error) {
	docsChan := make(chan domain.RawDocument)
	errsChan := make(chan error, 1)

	go func() {
		defer close(docsChan)
		defer close(errsChan)

		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			errsChan <- domain.ErrConnectorClosed
			return
		}
		c.mu.Unlock()

		// Create new cursor for full sync
		cursor := NewCursor()

		// List all accessible repositories
		repos, err := ListAllRepos(ctx, c.client)
		if err != nil {
			errsChan <- fmt.Errorf("list repos: %w", err)
			return
		}

		// Filter repositories
		repos = FilterRepos(repos, false, false)

		// Sync each repository.
		for _, repo := range repos {
			select {
			case <-ctx.Done():
				return
			default:
			}

			repoCursor := RepoCursor{}

			owner := repo.GetOwner().GetLogin()
			name := repo.GetName()

			// Fetch files if enabled.
			if c.config.HasContentType(ContentFiles) {
				docs, treeSHA, err := FetchFiles(ctx, c.client, repo, c.config)
				if err == nil || IsNotFound(err) {
					repoCursor.FilesTreeSHA = treeSHA
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case docsChan <- doc:
						}
					}
				}
			}

			// Fetch issues if enabled.
			if c.config.HasContentType(ContentIssues) {
				docs, latestUpdate, err := FetchIssues(ctx, c.client, repo, time.Time{})
				if err == nil || IsNotFound(err) {
					repoCursor.IssuesSince = latestUpdate
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case docsChan <- doc:
						}
					}
				}
			}

			// Fetch PRs if enabled.
			if c.config.HasContentType(ContentPRs) {
				docs, latestUpdate, err := FetchPullRequests(ctx, c.client, repo, time.Time{})
				if err == nil || IsNotFound(err) {
					repoCursor.PRsSince = latestUpdate
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case docsChan <- doc:
						}
					}
				}
			}

			// Fetch wiki if enabled.
			if c.config.HasContentType(ContentWikis) {
				docs, wikiSHA, err := FetchWikiPages(ctx, c.client, repo)
				if err == nil {
					repoCursor.WikiCommitSHA = wikiSHA
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case docsChan <- doc:
						}
					}
				}
			}

			// Save repo cursor.
			cursor.SetRepoCursor(owner, name, &repoCursor)
		}

		// Send completion with cursor
		errsChan <- &driven.SyncComplete{
			NewCursor: cursor.Encode(),
		}
	}()

	return docsChan, errsChan
}

// IncrementalSync fetches only changes since the last sync.
func (c *Connector) IncrementalSync(
	ctx context.Context, state domain.SyncState,
) (<-chan domain.RawDocumentChange, <-chan error) {
	changesChan := make(chan domain.RawDocumentChange)
	errsChan := make(chan error, 1)

	go func() {
		defer close(changesChan)
		defer close(errsChan)

		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			errsChan <- domain.ErrConnectorClosed
			return
		}
		c.mu.Unlock()

		// Decode cursor
		cursor, err := DecodeCursor(state.Cursor)
		if err != nil {
			errsChan <- fmt.Errorf("decode cursor: %w", err)
			return
		}

		// List all accessible repositories
		repos, err := ListAllRepos(ctx, c.client)
		if err != nil {
			errsChan <- fmt.Errorf("list repos: %w", err)
			return
		}

		repos = FilterRepos(repos, false, false)

		// Sync each repository.
		for _, repo := range repos {
			select {
			case <-ctx.Done():
				return
			default:
			}

			owner := repo.GetOwner().GetLogin()
			name := repo.GetName()
			branch := repo.GetDefaultBranch()
			repoCursor := cursor.GetRepoCursor(owner, name)

			// Fetch updated files if enabled.
			if c.config.HasContentType(ContentFiles) {
				// For files, we compare tree SHAs.
				currentTree, err := GetTree(ctx, c.client, owner, name, branch)
				if err == nil && currentTree.GetSHA() != repoCursor.FilesTreeSHA {
					// Tree changed, refetch all files (could optimize with diff).
					docs, treeSHA, err := FetchFiles(ctx, c.client, repo, c.config)
					if err == nil {
						repoCursor.FilesTreeSHA = treeSHA
						for _, doc := range docs {
							doc.SourceID = c.sourceID
							select {
							case <-ctx.Done():
								return
							case changesChan <- domain.RawDocumentChange{
								Type:     domain.ChangeUpdated,
								Document: doc,
							}:
							}
						}
					}
				}
			}

			// Fetch updated issues if enabled.
			if c.config.HasContentType(ContentIssues) {
				docs, latestUpdate, err := FetchIssues(ctx, c.client, repo, repoCursor.IssuesSince)
				if err == nil {
					if !latestUpdate.IsZero() {
						repoCursor.IssuesSince = latestUpdate
					}
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case changesChan <- domain.RawDocumentChange{
							Type:     domain.ChangeUpdated,
							Document: doc,
						}:
						}
					}
				}
			}

			// Fetch updated PRs if enabled.
			if c.config.HasContentType(ContentPRs) {
				docs, latestUpdate, err := FetchPullRequests(ctx, c.client, repo, repoCursor.PRsSince)
				if err == nil {
					if !latestUpdate.IsZero() {
						repoCursor.PRsSince = latestUpdate
					}
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case changesChan <- domain.RawDocumentChange{
							Type:     domain.ChangeUpdated,
							Document: doc,
						}:
						}
					}
				}
			}

			// Fetch updated wiki if enabled.
			if c.config.HasContentType(ContentWikis) {
				docs, wikiSHA, err := FetchWikiPages(ctx, c.client, repo)
				if err == nil && wikiSHA != repoCursor.WikiCommitSHA {
					repoCursor.WikiCommitSHA = wikiSHA
					for _, doc := range docs {
						doc.SourceID = c.sourceID
						select {
						case <-ctx.Done():
							return
						case changesChan <- domain.RawDocumentChange{
							Type:     domain.ChangeUpdated,
							Document: doc,
						}:
						}
					}
				}
			}

			// Update repo cursor.
			cursor.SetRepoCursor(owner, name, &repoCursor)
		}

		// Send completion with updated cursor
		errsChan <- &driven.SyncComplete{
			NewCursor: cursor.Encode(),
		}
	}()

	return changesChan, errsChan
}

// Watch is not supported for GitHub (no webhooks in CLI).
func (c *Connector) Watch(_ context.Context) (<-chan domain.RawDocumentChange, error) {
	return nil, domain.ErrNotImplemented
}

// GetAccountIdentifier fetches the GitHub username for the authenticated user.
func (c *Connector) GetAccountIdentifier(ctx context.Context, accessToken string) (string, error) {
	client := NewClientWithToken(ctx, accessToken)
	user, _, err := client.GitHub().Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	return user.GetLogin(), nil
}

// Close releases resources.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}
