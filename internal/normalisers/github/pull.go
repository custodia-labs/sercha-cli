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

// MIMETypeGitHubPull is the custom MIME type for GitHub pull requests.
const MIMETypeGitHubPull = "application/vnd.github.pull+json"

// Ensure PullNormaliser implements the interface.
var _ driven.Normaliser = (*PullNormaliser)(nil)

// PullNormaliser handles GitHub pull request documents.
type PullNormaliser struct{}

// NewPull creates a new GitHub pull request normaliser.
func NewPull() *PullNormaliser {
	return &PullNormaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *PullNormaliser) SupportedMIMETypes() []string {
	return []string{MIMETypeGitHubPull}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *PullNormaliser) SupportedConnectorTypes() []string {
	return []string{"github"} // GitHub-specific
}

// Priority returns the selection priority.
func (n *PullNormaliser) Priority() int {
	return 95 // Connector-specific priority
}

// PRContent represents the JSON content of a pull request.
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

// ReviewContent represents a review on a pull request.
type ReviewContent struct {
	Author      string    `json:"author"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// Normalise converts a GitHub PR document to a normalised document.
func (n *PullNormaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Parse JSON content
	var content PRContent
	if err := json.Unmarshal(raw.Content, &content); err != nil {
		return nil, fmt.Errorf("parse PR content: %w", err)
	}

	// Build normalised content with preserved authorship
	var sb strings.Builder

	// Header with metadata
	sb.WriteString(fmt.Sprintf("# Pull Request #%d: %s\n\n", content.Number, content.Title))

	// State badge
	state := content.State
	if content.Merged {
		state = "merged"
	} else if content.Draft {
		state = "draft"
	}
	sb.WriteString(fmt.Sprintf("**Author:** @%s | **State:** %s", content.Author, state))

	if len(content.Labels) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Labels:** %s", strings.Join(content.Labels, ", ")))
	}
	if len(content.Assignees) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Assignees:** @%s", strings.Join(content.Assignees, ", @")))
	}
	if len(content.Reviewers) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Reviewers:** @%s", strings.Join(content.Reviewers, ", @")))
	}
	sb.WriteString("\n\n")

	// Branch info
	sb.WriteString(fmt.Sprintf("**Branch:** `%s` → `%s`\n\n", content.HeadBranch, content.BaseBranch))

	// Timestamps
	sb.WriteString(fmt.Sprintf("*Created: %s | Updated: %s*\n\n",
		content.CreatedAt.Format("2006-01-02 15:04"),
		content.UpdatedAt.Format("2006-01-02 15:04")))

	// Changes summary
	sb.WriteString(fmt.Sprintf("**Changes:** +%d -%d in %d files\n\n",
		content.Additions, content.Deletions, content.ChangedFiles))

	// Description
	sb.WriteString("## Description\n\n")
	if content.Body != "" {
		sb.WriteString(content.Body)
	} else {
		sb.WriteString("*No description provided.*")
	}
	sb.WriteString("\n\n")

	// Reviews
	if len(content.Reviews) > 0 {
		sb.WriteString("## Reviews\n\n")
		for _, review := range content.Reviews {
			stateEmoji := getReviewStateEmoji(review.State)
			sb.WriteString(fmt.Sprintf("### %s @%s (%s)\n\n",
				stateEmoji,
				review.Author,
				review.SubmittedAt.Format("2006-01-02 15:04")))
			if review.Body != "" {
				sb.WriteString(review.Body)
				sb.WriteString("\n\n")
			}
		}
	}

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
	title := fmt.Sprintf("PR #%d: %s", content.Number, content.Title)

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
	doc.Metadata["format"] = "github_pull_request"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// getReviewStateEmoji returns an emoji for the review state.
func getReviewStateEmoji(state string) string {
	switch strings.ToUpper(state) {
	case "APPROVED":
		return "APPROVED"
	case "CHANGES_REQUESTED":
		return "CHANGES_REQUESTED"
	case "COMMENTED":
		return "COMMENTED"
	case "DISMISSED":
		return "DISMISSED"
	default:
		return state
	}
}
