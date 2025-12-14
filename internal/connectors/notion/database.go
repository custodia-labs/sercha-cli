package notion

import (
	"fmt"
	"strings"
	"time"

	"github.com/jomei/notionapi"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// DatabaseToRawDocument converts a Notion database to a RawDocument.
func DatabaseToRawDocument(db *notionapi.Database, sourceID string) *domain.RawDocument {
	title := extractDatabaseTitle(db)
	description := extractRichText(db.Description)

	// Build content from title and description
	var content strings.Builder
	content.WriteString("# ")
	content.WriteString(title)
	content.WriteString("\n\n")
	if description != "" {
		content.WriteString(description)
		content.WriteString("\n\n")
	}

	// Add property schema information
	content.WriteString("## Properties\n\n")
	for name, prop := range db.Properties {
		content.WriteString("- **")
		content.WriteString(name)
		content.WriteString("** (")
		content.WriteString(string(prop.GetType()))
		content.WriteString(")\n")
	}

	metadata := map[string]any{
		"database_id":      string(db.ID),
		"title":            title,
		"description":      description,
		"created_time":     db.CreatedTime.Format(time.RFC3339),
		"last_edited_time": db.LastEditedTime.Format(time.RFC3339),
		"created_by":       extractUserID(db.CreatedBy),
		"last_edited_by":   extractUserID(db.LastEditedBy),
		"archived":         db.Archived,
		"is_inline":        db.IsInline,
	}

	// Add URL if available
	if db.URL != "" {
		metadata["url"] = db.URL
	}

	// Add icon if available
	if db.Icon != nil {
		if db.Icon.Emoji != nil {
			metadata["icon"] = string(*db.Icon.Emoji)
		} else if db.Icon.External != nil {
			metadata["icon_url"] = db.Icon.External.URL
		}
	}

	// Add cover if available
	if db.Cover != nil {
		if db.Cover.External != nil {
			metadata["cover_url"] = db.Cover.External.URL
		} else if db.Cover.File != nil {
			metadata["cover_url"] = db.Cover.File.URL
		}
	}

	// Extract property schema
	propSchema := make(map[string]string)
	for name, prop := range db.Properties {
		propSchema[name] = string(prop.GetType())
	}
	metadata["property_schema"] = propSchema

	// Determine parent URI
	parentURI := buildDatabaseParentURI(db)

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("notion://databases/%s", db.ID),
		MIMEType:  MIMETypeNotionDB,
		Content:   []byte(content.String()),
		Metadata:  metadata,
		ParentURI: parentURI,
	}
}

// extractDatabaseTitle extracts the title from a database.
func extractDatabaseTitle(db *notionapi.Database) string {
	return extractRichText(db.Title)
}

// buildDatabaseParentURI constructs the parent URI for hierarchy tracking.
func buildDatabaseParentURI(db *notionapi.Database) *string {
	parent := db.Parent

	switch parent.Type { //nolint:exhaustive // Only relevant parent types handled
	case notionapi.ParentTypePageID:
		if parent.PageID != "" {
			uri := fmt.Sprintf("notion://pages/%s", parent.PageID)
			return &uri
		}
	case notionapi.ParentTypeBlockID:
		if parent.BlockID != "" {
			uri := fmt.Sprintf("notion://blocks/%s", parent.BlockID)
			return &uri
		}
	case notionapi.ParentTypeWorkspace:
		// Top-level databases have no parent
		return nil
	}

	return nil
}

// extractPropertyValues extracts values from page properties into a map.
//
//nolint:gocognit,gocyclo,funlen // Type switch over many property types is inherently complex
func extractPropertyValues(props notionapi.Properties) map[string]any {
	result := make(map[string]any)

	for name, prop := range props {
		switch p := prop.(type) {
		case *notionapi.TitleProperty:
			result[name] = extractRichText(p.Title)

		case *notionapi.RichTextProperty:
			result[name] = extractRichText(p.RichText)

		case *notionapi.NumberProperty:
			result[name] = p.Number

		case *notionapi.SelectProperty:
			if p.Select.Name != "" {
				result[name] = p.Select.Name
			}

		case *notionapi.MultiSelectProperty:
			names := make([]string, len(p.MultiSelect))
			for i, opt := range p.MultiSelect {
				names[i] = opt.Name
			}
			result[name] = names

		case *notionapi.DateProperty:
			if p.Date != nil {
				dateVal := map[string]any{
					"start": p.Date.Start.String(),
				}
				if p.Date.End != nil {
					dateVal["end"] = p.Date.End.String()
				}
				result[name] = dateVal
			}

		case *notionapi.CheckboxProperty:
			result[name] = p.Checkbox

		case *notionapi.URLProperty:
			result[name] = p.URL

		case *notionapi.EmailProperty:
			result[name] = p.Email

		case *notionapi.PhoneNumberProperty:
			result[name] = p.PhoneNumber

		case *notionapi.StatusProperty:
			if p.Status.Name != "" {
				result[name] = p.Status.Name
			}

		case *notionapi.PeopleProperty:
			people := make([]string, len(p.People))
			for i, person := range p.People {
				if person.Name != "" {
					people[i] = person.Name
				} else {
					people[i] = string(person.ID)
				}
			}
			result[name] = people

		case *notionapi.FilesProperty:
			files := make([]string, len(p.Files))
			for i, file := range p.Files {
				if file.File != nil && file.File.URL != "" {
					files[i] = file.File.URL
				} else if file.External != nil && file.External.URL != "" {
					files[i] = file.External.URL
				} else {
					files[i] = file.Name
				}
			}
			result[name] = files

		case *notionapi.RelationProperty:
			ids := make([]string, len(p.Relation))
			for i, rel := range p.Relation {
				ids[i] = string(rel.ID)
			}
			result[name] = ids

		case *notionapi.FormulaProperty:
			switch p.Formula.Type {
			case notionapi.FormulaTypeString:
				result[name] = p.Formula.String
			case notionapi.FormulaTypeNumber:
				result[name] = p.Formula.Number
			case notionapi.FormulaTypeBoolean:
				result[name] = p.Formula.Boolean
			case notionapi.FormulaTypeDate:
				if p.Formula.Date != nil {
					result[name] = p.Formula.Date.Start.String()
				}
			}

		case *notionapi.RollupProperty:
			switch p.Rollup.Type {
			case notionapi.RollupTypeNumber:
				result[name] = p.Rollup.Number
			case notionapi.RollupTypeDate:
				if p.Rollup.Date != nil {
					result[name] = p.Rollup.Date.Start.String()
				}
			case notionapi.RollupTypeArray:
				// Arrays are complex, store type info only
				result[name] = fmt.Sprintf("rollup_array(%d items)", len(p.Rollup.Array))
			}

		case *notionapi.CreatedTimeProperty:
			result[name] = p.CreatedTime.Format(time.RFC3339)

		case *notionapi.CreatedByProperty:
			result[name] = string(p.CreatedBy.ID)

		case *notionapi.LastEditedTimeProperty:
			result[name] = p.LastEditedTime.Format(time.RFC3339)

		case *notionapi.LastEditedByProperty:
			result[name] = string(p.LastEditedBy.ID)

		case *notionapi.UniqueIDProperty:
			if p.UniqueID.Prefix != nil {
				result[name] = fmt.Sprintf("%s-%d", *p.UniqueID.Prefix, p.UniqueID.Number)
			} else {
				result[name] = fmt.Sprintf("%d", p.UniqueID.Number)
			}
		}
	}

	return result
}
