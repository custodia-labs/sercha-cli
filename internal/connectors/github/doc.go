// Package github implements a connector for GitHub repositories.
//
// This connector indexes all repositories accessible to the authenticated user,
// including owned repositories, collaborator repositories, and organisation
// member repositories. Content types indexed include repository files, issues,
// pull requests, and wiki pages.
//
// # Architecture
//
// The connector follows the driven port pattern defined in [driven.Connector].
// It comprises the following components:
//
//   - Connector: orchestrates sync operations and manages lifecycle
//   - Client: handles GitHub API communication with rate limiting
//   - Config: parses and validates source configuration
//   - Cursor: tracks incremental sync state per repository
//
// # Authentication
//
// Two authentication methods are supported:
//
//   - Personal Access Tokens (PAT): classic or fine-grained tokens created at
//     github.com/settings/tokens. Requires 'repo' scope for private repositories.
//
//   - OAuth App: tokens obtained via the OAuth 2.0 authorisation code flow.
//     The application must be registered at github.com/settings/developers.
//
// Both methods provide 5,000 API requests per hour for authenticated users.
// Unauthenticated requests are limited to 60 per hour and are not supported.
//
// # Configuration
//
// Source configuration accepts the following keys:
//
//   - content_types: comma-separated list of content to index.
//     Valid values: files, issues, prs, wikis. Default: all types.
//
//   - file_patterns: comma-separated glob patterns for file filtering.
//     Example: "*.go,*.md". Default: all files.
//
// No repository specification is required. The connector automatically
// discovers and indexes all repositories accessible to the authenticated user.
//
// # Rate Limiting
//
// The connector implements a dual-strategy rate limiting approach:
//
//  1. Proactive throttling: a token bucket algorithm limits requests to
//     approximately 1.2 requests per second, staying well under the 5,000/hour
//     limit whilst maximising throughput.
//
//  2. Reactive handling: the connector monitors X-RateLimit-Remaining and
//     X-RateLimit-Reset headers. When limits are exhausted, it waits until
//     the reset time before continuing.
//
// Secondary rate limits (abuse detection) are handled with exponential backoff.
//
// # Sync Operations
//
// Full sync retrieves all content from all accessible repositories. For each
// repository, the connector:
//
//  1. Fetches the repository tree using the recursive Trees API
//  2. Retrieves blob content for each file matching configured patterns
//  3. Fetches issues and pull requests with their comments
//  4. Retrieves wiki pages if the repository has a wiki
//
// Incremental sync uses cursors to track sync state. The cursor stores:
//
//   - Tree SHA: detects file changes by comparing against the current HEAD
//   - Timestamps: filters issues and PRs updated since the last sync
//   - Wiki SHA: tracks wiki repository changes
//
// Each repository maintains independent cursor state, enabling partial syncs
// to resume from where they left off.
//
// # Document Structure
//
// Documents are emitted with the following URI patterns:
//
//   - Files: github://{owner}/{repo}/blob/{path}
//   - Issues: github://{owner}/{repo}/issues/{number}
//   - Pull Requests: github://{owner}/{repo}/pull/{number}
//   - Wiki Pages: github://{owner}/{repo}/wiki/{page}
//
// Metadata includes repository information, file paths, issue/PR state,
// labels, and timestamps.
//
// # Error Handling
//
// The connector distinguishes between recoverable and fatal errors:
//
//   - Rate limit errors: automatically retried after waiting
//   - Network errors: retried with exponential backoff
//   - Authentication errors: reported immediately as [domain.ErrAuthInvalid]
//   - Permission errors: logged and skipped (repository continues)
//
// # Limitations
//
//   - Binary files are not indexed (text content only)
//   - File size limit: 1MB per file (GitHub API constraint)
//   - Watch mode is not supported (no webhook integration in CLI)
//   - Private repository access requires appropriate token scopes
//
// # Example Usage
//
//	cfg, _ := github.ParseConfig(source)
//	connector := github.New(source.ID, cfg, tokenProvider)
//
//	if err := connector.Validate(ctx); err != nil {
//	    return err
//	}
//
//	docs, errs := connector.FullSync(ctx)
//	for doc := range docs {
//	    // Process document
//	}
package github
