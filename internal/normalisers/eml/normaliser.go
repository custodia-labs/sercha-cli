package eml

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// Normaliser handles EML (email) documents.
type Normaliser struct{}

// New creates a new EML normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{
		"message/rfc822",
	}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 50 // Generic MIME normaliser
}

// Normalise converts an EML document to a normalised document.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Parse the email message
	msg, err := mail.ReadMessage(bytes.NewReader(raw.Content))
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract headers
	subject := decodeHeader(msg.Header.Get("Subject"))
	from := decodeHeader(msg.Header.Get("From"))
	to := decodeHeader(msg.Header.Get("To"))
	date := msg.Header.Get("Date")

	// Extract body content
	body, err := extractBody(msg)
	if err != nil {
		return nil, err
	}

	// Build searchable content
	var content strings.Builder
	if from != "" {
		content.WriteString("From: ")
		content.WriteString(from)
		content.WriteString("\n")
	}
	if to != "" {
		content.WriteString("To: ")
		content.WriteString(to)
		content.WriteString("\n")
	}
	if date != "" {
		content.WriteString("Date: ")
		content.WriteString(date)
		content.WriteString("\n")
	}
	if subject != "" {
		content.WriteString("Subject: ")
		content.WriteString(subject)
		content.WriteString("\n")
	}
	content.WriteString("\n")
	content.WriteString(body)

	// Use subject as title, fall back to filename
	title := subject
	if title == "" {
		title = extractTitleFromURI(raw.URI)
	}

	// Build document
	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   strings.TrimSpace(content.String()),
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "eml"
	if from != "" {
		doc.Metadata["from"] = from
	}
	if to != "" {
		doc.Metadata["to"] = to
	}
	if date != "" {
		doc.Metadata["date"] = date
	}

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// decodeHeader decodes RFC 2047 encoded headers.
func decodeHeader(header string) string {
	if header == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		return header // Return original if decoding fails
	}
	return decoded
}

// extractBody extracts the text content from an email message.
func extractBody(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If we can't parse content type, try to read as plain text
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return "", domain.ErrInvalidInput
		}
		return string(body), nil
	}

	// Handle multipart messages
	if strings.HasPrefix(mediaType, "multipart/") {
		return extractMultipartBody(msg.Body, params["boundary"])
	}

	// Handle plain text or HTML
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return "", domain.ErrInvalidInput
	}

	if mediaType == "text/html" {
		return stripHTMLTags(string(body)), nil
	}

	return string(body), nil
}

// extractMultipartBody extracts text from multipart messages.
func extractMultipartBody(r io.Reader, boundary string) (string, error) {
	if boundary == "" {
		return "", nil
	}

	mr := multipart.NewReader(r, boundary)
	var textParts []string
	var htmlParts []string

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		partContentType := part.Header.Get("Content-Type")
		mediaType, params, parseErr := mime.ParseMediaType(partContentType)
		if parseErr != nil {
			mediaType = "application/octet-stream"
		}

		content, readErr := io.ReadAll(part)
		part.Close()
		if readErr != nil {
			continue
		}

		switch {
		case mediaType == "text/plain":
			textParts = append(textParts, string(content))
		case mediaType == "text/html":
			htmlParts = append(htmlParts, stripHTMLTags(string(content)))
		case strings.HasPrefix(mediaType, "multipart/"):
			// Recursively handle nested multipart
			nested, nestedErr := extractMultipartBody(bytes.NewReader(content), params["boundary"])
			if nestedErr == nil && nested != "" {
				textParts = append(textParts, nested)
			}
		}
	}

	// Prefer plain text over HTML
	if len(textParts) > 0 {
		return strings.Join(textParts, "\n"), nil
	}
	if len(htmlParts) > 0 {
		return strings.Join(htmlParts, "\n"), nil
	}

	return "", nil
}

// stripHTMLTags removes HTML tags for basic text extraction.
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false

	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Clean up whitespace
	text := result.String()
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// extractTitleFromURI extracts a title from the file URI.
func extractTitleFromURI(uri string) string {
	filename := filepath.Base(uri)
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")
	return filename
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
