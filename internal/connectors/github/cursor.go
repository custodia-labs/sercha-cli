package github

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// CursorVersion is the current cursor schema version.
const CursorVersion = 1

// Cursor tracks sync state across multiple repositories and content types.
type Cursor struct {
	// Version is the schema version for future migrations.
	Version int `json:"v"`

	// Repos maps repository full name (owner/repo) to its cursor state.
	Repos map[string]RepoCursor `json:"repos"`
}

// RepoCursor tracks sync state for a single repository.
type RepoCursor struct {
	// FilesTreeSHA is the Git tree SHA for the last indexed commit.
	FilesTreeSHA string `json:"files_sha,omitempty"`

	// IssuesSince is the timestamp of the last updated issue.
	IssuesSince time.Time `json:"issues_since,omitempty"`

	// PRsSince is the timestamp of the last updated PR.
	PRsSince time.Time `json:"prs_since,omitempty"`

	// WikiCommitSHA is the last indexed wiki commit SHA.
	WikiCommitSHA string `json:"wiki_sha,omitempty"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version: CursorVersion,
		Repos:   make(map[string]RepoCursor),
	}
}

// Encode serializes the cursor to a base64-encoded JSON string.
func (c *Cursor) Encode() string {
	if c == nil {
		return ""
	}
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor deserializes a cursor from a base64-encoded JSON string.
// Returns a new empty cursor if the input is empty or invalid.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return NewCursor(), nil
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	// Initialize maps if nil
	if cursor.Repos == nil {
		cursor.Repos = make(map[string]RepoCursor)
	}

	return &cursor, nil
}

// GetRepoCursor returns the cursor for a specific repository.
func (c *Cursor) GetRepoCursor(owner, repo string) RepoCursor {
	if c.Repos == nil {
		return RepoCursor{}
	}
	return c.Repos[owner+"/"+repo]
}

// SetRepoCursor sets the cursor for a specific repository.
func (c *Cursor) SetRepoCursor(owner, repo string, cursor *RepoCursor) {
	if c.Repos == nil {
		c.Repos = make(map[string]RepoCursor)
	}
	c.Repos[owner+"/"+repo] = *cursor
}

// UpdateFilesTreeSHA updates the files tree SHA for a repository.
func (c *Cursor) UpdateFilesTreeSHA(owner, repo, sha string) {
	rc := c.GetRepoCursor(owner, repo)
	rc.FilesTreeSHA = sha
	c.SetRepoCursor(owner, repo, &rc)
}

// UpdateIssuesSince updates the issues timestamp for a repository.
func (c *Cursor) UpdateIssuesSince(owner, repo string, t time.Time) {
	rc := c.GetRepoCursor(owner, repo)
	rc.IssuesSince = t
	c.SetRepoCursor(owner, repo, &rc)
}

// UpdatePRsSince updates the PRs timestamp for a repository.
func (c *Cursor) UpdatePRsSince(owner, repo string, t time.Time) {
	rc := c.GetRepoCursor(owner, repo)
	rc.PRsSince = t
	c.SetRepoCursor(owner, repo, &rc)
}

// UpdateWikiCommitSHA updates the wiki commit SHA for a repository.
func (c *Cursor) UpdateWikiCommitSHA(owner, repo, sha string) {
	rc := c.GetRepoCursor(owner, repo)
	rc.WikiCommitSHA = sha
	c.SetRepoCursor(owner, repo, &rc)
}

// RepoFullName returns the full repository name.
func RepoFullName(owner, repo string) string {
	return owner + "/" + repo
}
