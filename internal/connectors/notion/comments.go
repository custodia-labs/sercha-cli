package notion

import (
	"context"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// CommentFetcher fetches comments for pages.
type CommentFetcher struct {
	client   *Client
	pageSize int
}

// NewCommentFetcher creates a new comment fetcher.
func NewCommentFetcher(client *Client, pageSize int) *CommentFetcher {
	return &CommentFetcher{
		client:   client,
		pageSize: pageSize,
	}
}

// FetchComments retrieves all comments for a page or block.
func (f *CommentFetcher) FetchComments(ctx context.Context, blockID notionapi.BlockID) ([]string, error) {
	var comments []string
	var cursor notionapi.Cursor

	for {
		resp, err := f.client.GetComments(ctx, blockID, cursor, f.pageSize)
		if err != nil {
			// If we can't get comments, return empty (not critical)
			return comments, nil
		}

		for _, comment := range resp.Results {
			commentText := formatComment(&comment)
			if commentText != "" {
				comments = append(comments, commentText)
			}
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}

	return comments, nil
}

// formatComment formats a comment into a readable string.
func formatComment(comment *notionapi.Comment) string {
	var sb strings.Builder

	// Add author info if available
	if comment.CreatedBy.Name != "" {
		sb.WriteString(comment.CreatedBy.Name)
		sb.WriteString(": ")
	}

	// Extract comment text
	text := extractRichText(comment.RichText)
	sb.WriteString(text)

	// Add timestamp
	if !comment.CreatedTime.IsZero() {
		sb.WriteString(" (")
		sb.WriteString(comment.CreatedTime.Format(time.RFC3339))
		sb.WriteString(")")
	}

	return sb.String()
}

// FetchDiscussionComments retrieves comments from a discussion thread.
// This is used for inline comments on specific blocks.
func (f *CommentFetcher) FetchDiscussionComments(
	ctx context.Context, discussionID notionapi.BlockID,
) ([]string, error) {
	return f.FetchComments(ctx, discussionID)
}
