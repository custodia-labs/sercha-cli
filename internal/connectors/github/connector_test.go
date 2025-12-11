package github

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockTokenProvider implements driven.TokenProvider for testing.
type mockTokenProvider struct {
	token string
	err   error
}

func (p *mockTokenProvider) GetToken(_ context.Context) (string, error) {
	return p.token, p.err
}

func (p *mockTokenProvider) AuthorizationID() string {
	return "test-auth"
}

func (p *mockTokenProvider) AuthMethod() domain.AuthMethod {
	return domain.AuthMethodPAT
}

func (p *mockTokenProvider) IsAuthenticated() bool {
	return p.token != ""
}

func TestNew(t *testing.T) {
	t.Run("creates connector with valid parameters", func(t *testing.T) {
		cfg := &Config{
			ContentTypes: []ContentType{ContentFiles},
		}
		tokenProvider := &mockTokenProvider{token: "test-token"}

		connector := New("test-source", cfg, tokenProvider)

		require.NotNil(t, connector)
		assert.Equal(t, "test-source", connector.SourceID())
		assert.Equal(t, "github", connector.Type())
	})

	t.Run("creates connector with nil token provider", func(t *testing.T) {
		cfg := &Config{
			ContentTypes: AllContentTypes(),
		}

		connector := New("test-source", cfg, nil)

		require.NotNil(t, connector)
	})

	t.Run("implements Connector interface", func(t *testing.T) {
		cfg := &Config{}
		connector := New("test", cfg, nil)
		var _ driven.Connector = connector
	})
}

func TestConnector_Type(t *testing.T) {
	t.Run("returns github type", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		assert.Equal(t, "github", connector.Type())
	})

	t.Run("type is consistent across multiple calls", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		assert.Equal(t, connector.Type(), connector.Type())
	})
}

func TestConnector_SourceID(t *testing.T) {
	t.Run("returns correct source ID", func(t *testing.T) {
		connector := New("my-source-123", &Config{}, nil)

		assert.Equal(t, "my-source-123", connector.SourceID())
	})

	t.Run("handles empty source ID", func(t *testing.T) {
		connector := New("", &Config{}, nil)

		assert.Equal(t, "", connector.SourceID())
	})
}

func TestConnector_Capabilities(t *testing.T) {
	t.Run("returns expected capabilities", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		caps := connector.Capabilities()

		assert.True(t, caps.SupportsIncremental, "should support incremental sync")
		assert.True(t, caps.SupportsCursorReturn, "should support cursor return")
		assert.True(t, caps.SupportsValidation, "should support validation")
		assert.False(t, caps.SupportsWatch, "should not support watch")
		assert.False(t, caps.SupportsBinary, "should not support binary")
	})
}

func TestConnector_Close(t *testing.T) {
	t.Run("close succeeds", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		err := connector.Close()

		assert.NoError(t, err)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		err1 := connector.Close()
		err2 := connector.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func TestParseConfig(t *testing.T) {
	t.Run("parses valid config with all fields", func(t *testing.T) {
		source := domain.Source{
			ID:   "test-source",
			Type: "github",
			Config: map[string]string{
				"content_types": "files,issues,prs",
				"file_patterns": "*.go,*.md",
			},
		}

		cfg, err := ParseConfig(source)

		require.NoError(t, err)
		assert.Contains(t, cfg.ContentTypes, ContentFiles)
		assert.Contains(t, cfg.ContentTypes, ContentIssues)
		assert.Contains(t, cfg.ContentTypes, ContentPRs)
		assert.Contains(t, cfg.FilePatterns, "*.go")
		assert.Contains(t, cfg.FilePatterns, "*.md")
	})

	t.Run("parses minimal config with defaults", func(t *testing.T) {
		source := domain.Source{
			ID:     "test-source",
			Type:   "github",
			Config: map[string]string{},
		}

		cfg, err := ParseConfig(source)

		require.NoError(t, err)
		// Should default to all content types
		assert.Contains(t, cfg.ContentTypes, ContentFiles)
		assert.Contains(t, cfg.ContentTypes, ContentIssues)
		assert.Contains(t, cfg.ContentTypes, ContentPRs)
		assert.Contains(t, cfg.ContentTypes, ContentWikis)
	})

	t.Run("parses nil config with defaults", func(t *testing.T) {
		source := domain.Source{
			ID:     "test-source",
			Type:   "github",
			Config: nil,
		}

		cfg, err := ParseConfig(source)

		require.NoError(t, err)
		// Should default to all content types
		assert.Contains(t, cfg.ContentTypes, ContentFiles)
	})

	t.Run("parses wiki content type", func(t *testing.T) {
		source := domain.Source{
			ID:   "test-source",
			Type: "github",
			Config: map[string]string{
				"content_types": "wikis",
			},
		}

		cfg, err := ParseConfig(source)

		require.NoError(t, err)
		assert.Contains(t, cfg.ContentTypes, ContentWikis)
	})

	t.Run("returns error for invalid content types", func(t *testing.T) {
		source := domain.Source{
			ID:   "test-source",
			Type: "github",
			Config: map[string]string{
				"content_types": "files,invalid,issues",
			},
		}

		cfg, err := ParseConfig(source)

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, ErrConfigInvalidContentType)
	})
}

func TestCursor(t *testing.T) {
	t.Run("encodes and decodes cursor", func(t *testing.T) {
		original := &Cursor{
			Version: 1,
			Repos: map[string]RepoCursor{
				"myorg/myrepo": {
					FilesTreeSHA: "abc123",
					IssuesSince:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					PRsSince:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		}

		encoded := original.Encode()
		decoded, err := DecodeCursor(encoded)

		require.NoError(t, err)
		assert.Equal(t, original.Version, decoded.Version)
		assert.Equal(t, original.Repos["myorg/myrepo"].FilesTreeSHA, decoded.Repos["myorg/myrepo"].FilesTreeSHA)
	})

	t.Run("decode handles empty string", func(t *testing.T) {
		cursor, err := DecodeCursor("")

		require.NoError(t, err)
		assert.NotNil(t, cursor)
		assert.Empty(t, cursor.Repos)
	})

	t.Run("decode handles invalid base64", func(t *testing.T) {
		cursor, err := DecodeCursor("not-valid-base64!!!")

		assert.Error(t, err)
		assert.Nil(t, cursor)
	})

	t.Run("decode handles invalid JSON", func(t *testing.T) {
		invalidJSON := base64.StdEncoding.EncodeToString([]byte("not json"))

		cursor, err := DecodeCursor(invalidJSON)

		assert.Error(t, err)
		assert.Nil(t, cursor)
	})

	t.Run("GetRepoCursor returns zero value for unknown repo", func(t *testing.T) {
		cursor := &Cursor{
			Version: 1,
			Repos:   make(map[string]RepoCursor),
		}

		repoCursor := cursor.GetRepoCursor("unknown", "repo")

		assert.Equal(t, RepoCursor{}, repoCursor)
	})

	t.Run("SetRepoCursor updates existing repo", func(t *testing.T) {
		cursor := &Cursor{
			Version: 1,
			Repos:   make(map[string]RepoCursor),
		}

		cursor.SetRepoCursor("myorg", "myrepo", &RepoCursor{FilesTreeSHA: "sha1"})
		cursor.SetRepoCursor("myorg", "myrepo", &RepoCursor{FilesTreeSHA: "sha2"})

		assert.Equal(t, "sha2", cursor.Repos["myorg/myrepo"].FilesTreeSHA)
	})
}

func TestMatchesPatterns(t *testing.T) {
	t.Run("matches with empty patterns", func(t *testing.T) {
		assert.True(t, matchesPatterns("any/path.go", nil))
		assert.True(t, matchesPatterns("any/path.go", []string{}))
		assert.True(t, matchesPatterns("any/path.go", []string{"*"}))
	})

	t.Run("matches extension patterns", func(t *testing.T) {
		patterns := []string{"*.go", "*.md"}

		assert.True(t, matchesPatterns("cmd/main.go", patterns))
		assert.True(t, matchesPatterns("README.md", patterns))
		assert.False(t, matchesPatterns("package.json", patterns))
	})

	t.Run("matches against full path", func(t *testing.T) {
		patterns := []string{"cmd/*"}

		assert.True(t, matchesPatterns("cmd/main.go", patterns))
		assert.False(t, matchesPatterns("internal/main.go", patterns))
	})
}

func TestIsBinaryExtension(t *testing.T) {
	t.Run("identifies binary extensions", func(t *testing.T) {
		assert.True(t, isBinaryExtension("file.exe"))
		assert.True(t, isBinaryExtension("file.png"))
		assert.True(t, isBinaryExtension("file.pdf"))
		assert.True(t, isBinaryExtension("file.zip"))
	})

	t.Run("identifies non-binary extensions", func(t *testing.T) {
		assert.False(t, isBinaryExtension("file.go"))
		assert.False(t, isBinaryExtension("file.md"))
		assert.False(t, isBinaryExtension("file.txt"))
		assert.False(t, isBinaryExtension("file.json"))
	})

	t.Run("handles uppercase extensions", func(t *testing.T) {
		assert.True(t, isBinaryExtension("file.PNG"))
		assert.True(t, isBinaryExtension("file.PDF"))
	})

	t.Run("handles files without extension", func(t *testing.T) {
		assert.False(t, isBinaryExtension("Makefile"))
		assert.False(t, isBinaryExtension("Dockerfile"))
	})
}

func TestDetectFileMIMEType(t *testing.T) {
	t.Run("detects common MIME types", func(t *testing.T) {
		assert.Equal(t, "text/markdown", detectFileMIMEType("README.md"))
		assert.Equal(t, "text/x-go", detectFileMIMEType("main.go"))
		assert.Equal(t, "text/x-python", detectFileMIMEType("script.py"))
		assert.Equal(t, "text/yaml", detectFileMIMEType("config.yaml"))
		assert.Equal(t, "text/yaml", detectFileMIMEType("config.yml"))
	})

	t.Run("returns text/plain for unknown extensions", func(t *testing.T) {
		assert.Equal(t, "text/plain", detectFileMIMEType("file.unknown"))
		assert.Equal(t, "text/plain", detectFileMIMEType("Makefile"))
	})
}

func TestBuildFileURI(t *testing.T) {
	t.Run("builds correct URI", func(t *testing.T) {
		uri := buildFileURI("myorg", "myrepo", "main", "cmd/main.go")

		assert.Equal(t, "github://myorg/myrepo/blob/main/cmd/main.go", uri)
	})
}

func TestBuildWikiURI(t *testing.T) {
	t.Run("builds correct wiki URI", func(t *testing.T) {
		uri := buildWikiURI("myorg", "myrepo", "Home")

		assert.Equal(t, "github://myorg/myrepo/wiki/Home", uri)
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("creates rate limiter with defaults", func(t *testing.T) {
		rl := NewRateLimiter()

		require.NotNil(t, rl)
		assert.Equal(t, GitHubRateLimit, rl.Limit())
		assert.Equal(t, GitHubRateLimit, rl.Remaining())
	})

	t.Run("updates from response headers", func(t *testing.T) {
		rl := NewRateLimiter()
		resetTime := time.Now().Add(1 * time.Hour).Unix()

		resp := &http.Response{
			Header: http.Header{
				"X-Ratelimit-Remaining": []string{"100"},
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Reset":     []string{string(rune(resetTime))},
			},
		}

		rl.UpdateFromResponse(resp)

		assert.Equal(t, 100, rl.Remaining())
		assert.Equal(t, 5000, rl.Limit())
	})

	t.Run("wait respects context cancellation", func(t *testing.T) {
		rl := NewRateLimiter()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := rl.Wait(ctx)

		assert.Error(t, err)
	})
}

func TestConnector_Validate_NoToken(t *testing.T) {
	t.Run("validation fails with unauthenticated provider", func(t *testing.T) {
		cfg := &Config{}
		tokenProvider := &mockTokenProvider{token: ""}
		connector := New("test", cfg, tokenProvider)

		err := connector.Validate(context.Background())

		assert.Error(t, err)
	})
}

// Tests for Config.HasContentType
func TestConfig_HasContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentTypes []ContentType
		check        ContentType
		want         bool
	}{
		{
			name:         "has files content type",
			contentTypes: []ContentType{ContentFiles, ContentIssues},
			check:        ContentFiles,
			want:         true,
		},
		{
			name:         "has issues content type",
			contentTypes: []ContentType{ContentFiles, ContentIssues},
			check:        ContentIssues,
			want:         true,
		},
		{
			name:         "does not have wikis content type",
			contentTypes: []ContentType{ContentFiles, ContentIssues},
			check:        ContentWikis,
			want:         false,
		},
		{
			name:         "empty content types returns false",
			contentTypes: []ContentType{},
			check:        ContentFiles,
			want:         false,
		},
		{
			name:         "has all content types",
			contentTypes: AllContentTypes(),
			check:        ContentPRs,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ContentTypes: tt.contentTypes}
			got := cfg.HasContentType(tt.check)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for Client.GitHub
func TestClient_GitHub(t *testing.T) {
	t.Run("returns nil when client not initialized", func(t *testing.T) {
		tokenProvider := &mockTokenProvider{token: "test-token"}
		client := NewClient(tokenProvider)

		gh := client.GitHub()

		assert.Nil(t, gh)
	})

	t.Run("returns client after initialization", func(t *testing.T) {
		ctx := context.Background()
		token := "test-token"
		client := NewClientWithToken(ctx, token)

		gh := client.GitHub()

		assert.NotNil(t, gh)
	})
}

// Tests for Client.TokenProvider
func TestClient_TokenProvider(t *testing.T) {
	t.Run("returns the token provider", func(t *testing.T) {
		tokenProvider := &mockTokenProvider{token: "test-token"}
		client := NewClient(tokenProvider)

		tp := client.TokenProvider()

		assert.Equal(t, tokenProvider, tp)
	})

	t.Run("returns nil when no token provider", func(t *testing.T) {
		ctx := context.Background()
		client := NewClientWithToken(ctx, "token")

		tp := client.TokenProvider()

		assert.Nil(t, tp)
	})
}

// Tests for Client.RateLimiter
func TestClient_RateLimiter(t *testing.T) {
	t.Run("returns the rate limiter", func(t *testing.T) {
		tokenProvider := &mockTokenProvider{token: "test-token"}
		client := NewClient(tokenProvider)

		rl := client.RateLimiter()

		assert.NotNil(t, rl)
		assert.Equal(t, GitHubRateLimit, rl.Limit())
	})

	t.Run("rate limiter is initialized on creation", func(t *testing.T) {
		ctx := context.Background()
		client := NewClientWithToken(ctx, "token")

		rl := client.RateLimiter()

		require.NotNil(t, rl)
		assert.Equal(t, GitHubRateLimit, rl.Remaining())
	})
}

// Tests for wrapError
func TestClient_WrapError(t *testing.T) {
	tokenProvider := &mockTokenProvider{token: "test-token"}
	client := NewClient(tokenProvider)

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := client.wrapError(nil, "test operation")

		assert.NoError(t, err)
	})

	t.Run("wraps github ErrorResponse as APIError", func(t *testing.T) {
		testURL, _ := url.Parse("https://api.github.com/repos/test/repo")
		ghErr := &gh.ErrorResponse{
			Response: &http.Response{
				StatusCode: 404,
				Request: &http.Request{
					URL: testURL,
				},
			},
			Message: "Not Found",
		}

		err := client.wrapError(ghErr, "get repo")

		require.Error(t, err)
		var apiErr *APIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 404, apiErr.StatusCode)
		assert.Equal(t, "Not Found", apiErr.Message)
	})

	t.Run("wraps github RateLimitError", func(t *testing.T) {
		ghErr := &gh.RateLimitError{
			Rate: gh.Rate{
				Limit:     5000,
				Remaining: 0,
				Reset:     gh.Timestamp{Time: time.Now().Add(1 * time.Hour)},
			},
		}

		err := client.wrapError(ghErr, "list repos")

		require.Error(t, err)
		var rateLimitErr *RateLimitError
		assert.True(t, errors.As(err, &rateLimitErr))
		// wrapError uses client.rateLimiter values (defaults to GitHubRateLimit)
		// not the values from the GitHub error itself
		assert.Equal(t, GitHubRateLimit, rateLimitErr.Remaining)
		assert.Equal(t, GitHubRateLimit, rateLimitErr.Limit)
	})

	t.Run("wraps generic error with operation", func(t *testing.T) {
		genericErr := errors.New("network error")

		err := client.wrapError(genericErr, "fetch data")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch data")
		assert.Contains(t, err.Error(), "network error")
	})
}

// Tests for buildIssueURI
func TestBuildIssueURI(t *testing.T) {
	tests := []struct {
		name   string
		owner  string
		repo   string
		number int
		want   string
	}{
		{
			name:   "basic issue URI",
			owner:  "octocat",
			repo:   "hello-world",
			number: 123,
			want:   "github://octocat/hello-world/issues/123",
		},
		{
			name:   "issue with org owner",
			owner:  "github",
			repo:   "docs",
			number: 1,
			want:   "github://github/docs/issues/1",
		},
		{
			name:   "issue with large number",
			owner:  "test",
			repo:   "repo",
			number: 999999,
			want:   "github://test/repo/issues/999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildIssueURI(tt.owner, tt.repo, tt.number)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for buildPRURI
func TestBuildPRURI(t *testing.T) {
	tests := []struct {
		name   string
		owner  string
		repo   string
		number int
		want   string
	}{
		{
			name:   "basic PR URI",
			owner:  "octocat",
			repo:   "hello-world",
			number: 456,
			want:   "github://octocat/hello-world/pull/456",
		},
		{
			name:   "PR with org owner",
			owner:  "microsoft",
			repo:   "vscode",
			number: 100,
			want:   "github://microsoft/vscode/pull/100",
		},
		{
			name:   "PR with special characters in names",
			owner:  "test-org",
			repo:   "my.repo",
			number: 5,
			want:   "github://test-org/my.repo/pull/5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPRURI(tt.owner, tt.repo, tt.number)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for NewClient
func TestNewClient(t *testing.T) {
	t.Run("creates client with valid token provider", func(t *testing.T) {
		tokenProvider := &mockTokenProvider{token: "test-token"}

		client := NewClient(tokenProvider)

		require.NotNil(t, client)
		assert.NotNil(t, client.RateLimiter())
		assert.Equal(t, tokenProvider, client.TokenProvider())
		assert.Nil(t, client.GitHub()) // Not initialized yet
	})

	t.Run("creates client with nil token provider", func(t *testing.T) {
		client := NewClient(nil)

		require.NotNil(t, client)
		assert.NotNil(t, client.RateLimiter())
		assert.Nil(t, client.TokenProvider())
	})
}

// Tests for NewClientWithToken
func TestNewClientWithToken(t *testing.T) {
	t.Run("creates client with valid token", func(t *testing.T) {
		ctx := context.Background()
		token := "ghp_test_token_123"

		client := NewClientWithToken(ctx, token)

		require.NotNil(t, client)
		assert.NotNil(t, client.GitHub())
		assert.NotNil(t, client.RateLimiter())
	})

	t.Run("creates client with empty token", func(t *testing.T) {
		ctx := context.Background()

		client := NewClientWithToken(ctx, "")

		require.NotNil(t, client)
		assert.NotNil(t, client.GitHub())
	})
}

// Tests for NewClientWithHTTPClient
func TestNewClientWithHTTPClient(t *testing.T) {
	t.Run("creates client with custom http client", func(t *testing.T) {
		httpClient := &http.Client{Timeout: 10 * time.Second}

		client := NewClientWithHTTPClient(httpClient)

		require.NotNil(t, client)
		assert.NotNil(t, client.GitHub())
		assert.NotNil(t, client.RateLimiter())
	})

	t.Run("creates client with nil http client", func(t *testing.T) {
		client := NewClientWithHTTPClient(nil)

		require.NotNil(t, client)
		assert.NotNil(t, client.GitHub())
	})
}

// Tests for error helper functions
func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError with 404 status",
			err:  &APIError{StatusCode: 404, Message: "Not Found"},
			want: true,
		},
		{
			name: "APIError with 403 status",
			err:  &APIError{StatusCode: 403, Message: "Forbidden"},
			want: false,
		},
		{
			name: "ErrRepoNotFound",
			err:  ErrRepoNotFound,
			want: true,
		},
		{
			name: "ErrBranchNotFound",
			err:  ErrBranchNotFound,
			want: true,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "RateLimitError",
			err:  &RateLimitError{Limit: 5000, Remaining: 0},
			want: true,
		},
		{
			name: "APIError",
			err:  &APIError{StatusCode: 429},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimited(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError with 401 status",
			err:  &APIError{StatusCode: 401, Message: "Unauthorized"},
			want: true,
		},
		{
			name: "APIError with 403 status",
			err:  &APIError{StatusCode: 403, Message: "Forbidden"},
			want: false,
		},
		{
			name: "APIError with 404 status",
			err:  &APIError{StatusCode: 404},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("auth failed"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnauthorized(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsForbidden(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError with 403 status",
			err:  &APIError{StatusCode: 403, Message: "Forbidden"},
			want: true,
		},
		{
			name: "APIError with 401 status",
			err:  &APIError{StatusCode: 401},
			want: false,
		},
		{
			name: "APIError with 404 status",
			err:  &APIError{StatusCode: 404},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("access denied"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsForbidden(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for error types Error() methods
func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantString string
	}{
		{
			name: "complete error",
			err: &APIError{
				StatusCode: 404,
				Message:    "Not Found",
				URL:        "https://api.github.com/repos/test/repo",
			},
			wantString: "github: API error 404: Not Found (URL: https://api.github.com/repos/test/repo)",
		},
		{
			name: "error with empty message",
			err: &APIError{
				StatusCode: 500,
				Message:    "",
				URL:        "https://api.github.com/test",
			},
			wantString: "github: API error 500:  (URL: https://api.github.com/test)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.wantString, got)
		})
	}
}

func TestRateLimitError_Error(t *testing.T) {
	t.Run("formats error message with reset time", func(t *testing.T) {
		resetTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		err := &RateLimitError{
			ResetAt:   resetTime,
			Remaining: 0,
			Limit:     5000,
		}

		got := err.Error()

		assert.Contains(t, got, "rate limit exceeded")
		assert.Contains(t, got, "2024-01-01T12:00:00Z")
	})
}

// Tests for AllContentTypes
func TestAllContentTypes(t *testing.T) {
	t.Run("returns all supported content types", func(t *testing.T) {
		types := AllContentTypes()

		assert.Len(t, types, 4)
		assert.Contains(t, types, ContentFiles)
		assert.Contains(t, types, ContentIssues)
		assert.Contains(t, types, ContentPRs)
		assert.Contains(t, types, ContentWikis)
	})

	t.Run("returns unique content types", func(t *testing.T) {
		types := AllContentTypes()

		seen := make(map[ContentType]bool)
		for _, ct := range types {
			assert.False(t, seen[ct], "duplicate content type: %s", ct)
			seen[ct] = true
		}
	})
}

// Tests for Connector.Watch
func TestConnector_Watch(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		connector := New("test", &Config{}, nil)

		ch, err := connector.Watch(context.Background())

		assert.Nil(t, ch)
		assert.ErrorIs(t, err, domain.ErrNotImplemented)
	})
}
