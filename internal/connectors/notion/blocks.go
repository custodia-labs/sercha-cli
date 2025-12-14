package notion

import (
	"context"
	"strings"

	"github.com/jomei/notionapi"
)

// BlockExtractor extracts text content from Notion blocks recursively.
type BlockExtractor struct {
	client   *Client
	maxDepth int
	pageSize int
}

// NewBlockExtractor creates a new block extractor.
func NewBlockExtractor(client *Client, maxDepth, pageSize int) *BlockExtractor {
	return &BlockExtractor{
		client:   client,
		maxDepth: maxDepth,
		pageSize: pageSize,
	}
}

// ExtractContent extracts all text content from a page or block.
func (e *BlockExtractor) ExtractContent(ctx context.Context, blockID notionapi.BlockID) (string, error) {
	return e.extractContentRecursive(ctx, blockID, 0)
}

// extractContentRecursive fetches and extracts content recursively.
func (e *BlockExtractor) extractContentRecursive(
	ctx context.Context, blockID notionapi.BlockID, depth int,
) (string, error) {
	if depth > e.maxDepth {
		return "", nil
	}

	var content strings.Builder
	var cursor notionapi.Cursor

	for {
		resp, err := e.client.GetBlockChildren(ctx, blockID, cursor, e.pageSize)
		if err != nil {
			// If we can't get blocks, return what we have
			return content.String(), nil
		}

		for _, block := range resp.Results {
			// Extract text from this block
			blockText := extractBlockText(block)
			content.WriteString(blockText)

			// Recursively extract children if the block has them
			if block.GetHasChildren() {
				childContent, err := e.extractContentRecursive(ctx, block.GetID(), depth+1)
				if err == nil && childContent != "" {
					content.WriteString(childContent)
				}
			}
		}

		if !resp.HasMore {
			break
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}

	return content.String(), nil
}

// extractBlockText extracts text from a single block based on its type.
//
//nolint:gocognit,gocyclo,funlen // Type switch over many block types is inherently complex
func extractBlockText(block notionapi.Block) string {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return extractRichText(b.Paragraph.RichText) + "\n\n"

	case *notionapi.Heading1Block:
		return "# " + extractRichText(b.Heading1.RichText) + "\n\n"

	case *notionapi.Heading2Block:
		return "## " + extractRichText(b.Heading2.RichText) + "\n\n"

	case *notionapi.Heading3Block:
		return "### " + extractRichText(b.Heading3.RichText) + "\n\n"

	case *notionapi.BulletedListItemBlock:
		return "- " + extractRichText(b.BulletedListItem.RichText) + "\n"

	case *notionapi.NumberedListItemBlock:
		return "1. " + extractRichText(b.NumberedListItem.RichText) + "\n"

	case *notionapi.ToDoBlock:
		checkbox := "[ ] "
		if b.ToDo.Checked {
			checkbox = "[x] "
		}
		return checkbox + extractRichText(b.ToDo.RichText) + "\n"

	case *notionapi.ToggleBlock:
		return extractRichText(b.Toggle.RichText) + "\n"

	case *notionapi.CodeBlock:
		lang := b.Code.Language
		return "```" + lang + "\n" + extractRichText(b.Code.RichText) + "\n```\n\n"

	case *notionapi.QuoteBlock:
		return "> " + extractRichText(b.Quote.RichText) + "\n\n"

	case *notionapi.CalloutBlock:
		icon := ""
		if b.Callout.Icon != nil && b.Callout.Icon.Emoji != nil {
			icon = string(*b.Callout.Icon.Emoji) + " "
		}
		return icon + extractRichText(b.Callout.RichText) + "\n\n"

	case *notionapi.DividerBlock:
		return "---\n\n"

	case *notionapi.TableOfContentsBlock:
		return "[Table of Contents]\n\n"

	case *notionapi.BookmarkBlock:
		url := ""
		if b.Bookmark.URL != "" {
			url = b.Bookmark.URL
		}
		caption := extractRichText(b.Bookmark.Caption)
		if caption != "" {
			return "[" + caption + "](" + url + ")\n\n"
		}
		return url + "\n\n"

	case *notionapi.LinkPreviewBlock:
		return b.LinkPreview.URL + "\n\n"

	case *notionapi.EquationBlock:
		return "$" + b.Equation.Expression + "$\n\n"

	case *notionapi.ImageBlock:
		caption := extractRichText(b.Image.Caption)
		url := getFileURL(b.Image.File, b.Image.External)
		if caption != "" {
			return "![" + caption + "](" + url + ")\n\n"
		}
		return "![Image](" + url + ")\n\n"

	case *notionapi.VideoBlock:
		caption := extractRichText(b.Video.Caption)
		url := getFileURL(b.Video.File, b.Video.External)
		if caption != "" {
			return "[Video: " + caption + "](" + url + ")\n\n"
		}
		return "[Video](" + url + ")\n\n"

	case *notionapi.FileBlock:
		caption := extractRichText(b.File.Caption)
		url := getFileURL(b.File.File, b.File.External)
		if caption != "" {
			return "[File: " + caption + "](" + url + ")\n\n"
		}
		return "[File](" + url + ")\n\n"

	case *notionapi.PdfBlock:
		caption := extractRichText(b.Pdf.Caption)
		url := getFileURL(b.Pdf.File, b.Pdf.External)
		if caption != "" {
			return "[PDF: " + caption + "](" + url + ")\n\n"
		}
		return "[PDF](" + url + ")\n\n"

	case *notionapi.AudioBlock:
		caption := extractRichText(b.Audio.Caption)
		url := getFileURL(b.Audio.File, b.Audio.External)
		if caption != "" {
			return "[Audio: " + caption + "](" + url + ")\n\n"
		}
		return "[Audio](" + url + ")\n\n"

	case *notionapi.EmbedBlock:
		caption := extractRichText(b.Embed.Caption)
		if caption != "" {
			return "[Embed: " + caption + "](" + b.Embed.URL + ")\n\n"
		}
		return "[Embed](" + b.Embed.URL + ")\n\n"

	case *notionapi.ChildPageBlock:
		return "[Child Page: " + b.ChildPage.Title + "]\n\n"

	case *notionapi.ChildDatabaseBlock:
		return "[Child Database: " + b.ChildDatabase.Title + "]\n\n"

	case *notionapi.SyncedBlock:
		// Synced blocks contain duplicate content - skip to avoid duplicates
		return ""

	case *notionapi.ColumnListBlock, *notionapi.ColumnBlock:
		// Columns are structural, content comes from children
		return ""

	case *notionapi.TableBlock:
		// Table header info only; rows come from children
		return ""

	case *notionapi.TableRowBlock:
		cells := make([]string, len(b.TableRow.Cells))
		for i, cell := range b.TableRow.Cells {
			cells[i] = extractRichText(cell)
		}
		return "| " + strings.Join(cells, " | ") + " |\n"

	case *notionapi.BreadcrumbBlock, *notionapi.TemplateBlock, *notionapi.LinkToPageBlock:
		// Structural/navigation blocks, no text content
		return ""

	default:
		return ""
	}
}

// extractRichText concatenates rich text segments into plain text.
func extractRichText(richText []notionapi.RichText) string {
	var sb strings.Builder
	for _, rt := range richText {
		sb.WriteString(rt.PlainText)
	}
	return sb.String()
}

// getFileURL extracts the URL from a file reference.
func getFileURL(file, external *notionapi.FileObject) string {
	if file != nil && file.URL != "" {
		return file.URL
	}
	if external != nil && external.URL != "" {
		return external.URL
	}
	return ""
}

// ExtractTitle extracts the title from rich text (used for page titles).
func ExtractTitle(richText []notionapi.RichText) string {
	return extractRichText(richText)
}
