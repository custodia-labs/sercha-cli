package notion

import (
	"fmt"
	"time"

	"github.com/jomei/notionapi"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MIME types for Notion documents.
const (
	MIMETypeNotionPage   = "application/vnd.notion.page+json"
	MIMETypeNotionDB     = "application/vnd.notion.database+json"
	MIMETypeNotionDBItem = "application/vnd.notion.database-item+json"
)

// PageToRawDocument converts a Notion page to a RawDocument.
func PageToRawDocument(page *notionapi.Page, content, sourceID string, comments []string) *domain.RawDocument {
	title := extractPageTitle(page)

	metadata := map[string]any{
		"page_id":          string(page.ID),
		"title":            title,
		"created_time":     page.CreatedTime.Format(time.RFC3339),
		"last_edited_time": page.LastEditedTime.Format(time.RFC3339),
		"created_by":       extractUserID(page.CreatedBy),
		"last_edited_by":   extractUserID(page.LastEditedBy),
		"archived":         page.Archived,
	}

	// Add URL if available
	if page.URL != "" {
		metadata["url"] = page.URL
	}

	// Add icon if available
	if page.Icon != nil {
		if page.Icon.Emoji != nil {
			metadata["icon"] = string(*page.Icon.Emoji)
		} else if page.Icon.External != nil {
			metadata["icon_url"] = page.Icon.External.URL
		}
	}

	// Add cover if available
	if page.Cover != nil {
		if page.Cover.External != nil {
			metadata["cover_url"] = page.Cover.External.URL
		} else if page.Cover.File != nil {
			metadata["cover_url"] = page.Cover.File.URL
		}
	}

	// Add comments if any
	if len(comments) > 0 {
		metadata["comments"] = comments
	}

	// Determine parent URI
	parentURI := buildPageParentURI(page)

	// Determine MIME type (database item vs regular page)
	mimeType := MIMETypeNotionPage
	if page.Parent.DatabaseID != "" {
		mimeType = MIMETypeNotionDBItem
		metadata["database_id"] = string(page.Parent.DatabaseID)
	}

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("notion://pages/%s", page.ID),
		MIMEType:  mimeType,
		Content:   []byte(content),
		Metadata:  metadata,
		ParentURI: parentURI,
	}
}

// extractPageTitle extracts the title from a page's properties.
func extractPageTitle(page *notionapi.Page) string {
	for _, prop := range page.Properties {
		if p, ok := prop.(*notionapi.TitleProperty); ok {
			return extractRichText(p.Title)
		}
	}
	return "Untitled"
}

// extractUserID extracts a user ID from a User.
func extractUserID(user notionapi.User) string {
	return string(user.ID)
}

// buildPageParentURI constructs the parent URI for hierarchy tracking.
func buildPageParentURI(page *notionapi.Page) *string {
	parent := page.Parent

	switch parent.Type {
	case notionapi.ParentTypePageID:
		if parent.PageID != "" {
			uri := fmt.Sprintf("notion://pages/%s", parent.PageID)
			return &uri
		}
	case notionapi.ParentTypeDatabaseID:
		if parent.DatabaseID != "" {
			uri := fmt.Sprintf("notion://databases/%s", parent.DatabaseID)
			return &uri
		}
	case notionapi.ParentTypeBlockID:
		if parent.BlockID != "" {
			// Blocks are part of pages, treat as page reference
			uri := fmt.Sprintf("notion://blocks/%s", parent.BlockID)
			return &uri
		}
	case notionapi.ParentTypeWorkspace:
		// Top-level pages have no parent
		return nil
	}

	return nil
}

// PageWithProperties converts a page with its database properties to a RawDocument.
// This is used for database items where properties contain structured data.
func PageWithProperties(page *notionapi.Page, content, sourceID string, comments []string) *domain.RawDocument {
	doc := PageToRawDocument(page, content, sourceID, comments)

	// Extract all properties as metadata
	props := extractPropertyValues(page.Properties)
	for k, v := range props {
		// Prefix property keys to avoid conflicts
		doc.Metadata["prop_"+k] = v
	}

	return doc
}
