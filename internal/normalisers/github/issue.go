package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// MIMETypeGitHubIssue is the custom MIME type for GitHub issues.
const MIMETypeGitHubIssue = "application/vnd.github.issue+json"

// Ensure IssueNormaliser implements the interface.
var _ driven.Normaliser = (*IssueNormaliser)(nil)

// IssueNormaliser handles GitHub issue documents.
type IssueNormaliser struct{}

// NewIssue creates a new GitHub issue normaliser.
func NewIssue() *IssueNormaliser {
	return &IssueNormaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *IssueNormaliser) SupportedMIMETypes() []string {
	return []string{MIMETypeGitHubIssue}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *IssueNormaliser) SupportedConnectorTypes() []string {
	return []string{"github"} // GitHub-specific
}

// Priority returns the selection priority.
func (n *IssueNormaliser) Priority() int {
	return 95 // Connector-specific priority
}

// IssueContent represents the JSON content of an issue.
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

// CommentContent represents a comment on an issue.
type CommentContent struct {
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Normalise converts a GitHub issue document to a normalised document.
func (n *IssueNormaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Parse JSON content
	var content IssueContent
	if err := json.Unmarshal(raw.Content, &content); err != nil {
		return nil, fmt.Errorf("parse issue content: %w", err)
	}

	// Build normalised content with preserved authorship
	var sb strings.Builder

	// Header with metadata
	sb.WriteString(fmt.Sprintf("# Issue #%d: %s\n\n", content.Number, content.Title))
	sb.WriteString(fmt.Sprintf("**Author:** @%s | **State:** %s", content.Author, content.State))
	if len(content.Labels) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Labels:** %s", strings.Join(content.Labels, ", ")))
	}
	if len(content.Assignees) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Assignees:** @%s", strings.Join(content.Assignees, ", @")))
	}
	if content.Milestone != "" {
		sb.WriteString(fmt.Sprintf(" | **Milestone:** %s", content.Milestone))
	}
	sb.WriteString("\n\n")

	// Timestamps
	sb.WriteString(fmt.Sprintf("*Created: %s | Updated: %s*\n\n",
		content.CreatedAt.Format("2006-01-02 15:04"),
		content.UpdatedAt.Format("2006-01-02 15:04")))

	// Description
	sb.WriteString("## Description\n\n")
	if content.Body != "" {
		sb.WriteString(content.Body)
	} else {
		sb.WriteString("*No description provided.*")
	}
	sb.WriteString("\n\n")

	// Comments
	if len(content.Comments) > 0 {
		sb.WriteString("## Comments\n\n")
		for _, comment := range content.Comments {
			sb.WriteString(fmt.Sprintf("### @%s (%s)\n\n%s\n\n",
				comment.Author,
				comment.CreatedAt.Format("2006-01-02 15:04"),
				comment.Body))
		}
	}

	// Build title
	title := fmt.Sprintf("Issue #%d: %s", content.Number, content.Title)

	// Build document
	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   sb.String(),
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add normaliser info to metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "github_issue"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// copyMetadata creates a shallow copy of metadata.
func copyMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
