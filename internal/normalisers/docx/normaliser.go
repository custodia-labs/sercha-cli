package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// Normaliser handles DOCX documents.
type Normaliser struct{}

// New creates a new DOCX normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
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

// Normalise converts a DOCX document to a normalised document.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Open as ZIP archive
	reader, err := zip.NewReader(bytes.NewReader(raw.Content), int64(len(raw.Content)))
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract text content from document.xml
	content, err := extractDocumentText(reader)
	if err != nil {
		return nil, err
	}

	// Extract title from core.xml or fall back to filename
	title := extractTitle(reader, raw.URI)

	// Build document
	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   content,
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "docx"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// extractDocumentText extracts text from word/document.xml.
func extractDocumentText(reader *zip.Reader) (string, error) {
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return "", domain.ErrInvalidInput
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", domain.ErrInvalidInput
		}

		return parseDocumentXML(content), nil
	}
	return "", nil
}

// documentXML represents the structure of word/document.xml.
type documentXML struct {
	Body struct {
		Paragraphs []paragraph `xml:"p"`
	} `xml:"body"`
}

type paragraph struct {
	Runs []run `xml:"r"`
}

type run struct {
	Text []textElement `xml:"t"`
}

type textElement struct {
	Content string `xml:",chardata"`
}

// parseDocumentXML extracts text content from the document XML.
func parseDocumentXML(content []byte) string {
	var doc documentXML
	if err := xml.Unmarshal(content, &doc); err != nil {
		return ""
	}

	var result strings.Builder
	for i, para := range doc.Body.Paragraphs {
		if i > 0 {
			result.WriteString("\n")
		}
		for _, run := range para.Runs {
			for _, text := range run.Text {
				result.WriteString(text.Content)
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// coreXML represents the structure of docProps/core.xml.
type coreXML struct {
	Title string `xml:"title"`
}

// extractTitle extracts the title from docProps/core.xml or falls back to filename.
func extractTitle(reader *zip.Reader, uri string) string {
	for _, file := range reader.File {
		if file.Name != "docProps/core.xml" {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			break
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			break
		}

		var core coreXML
		if err := xml.Unmarshal(content, &core); err == nil && core.Title != "" {
			return strings.TrimSpace(core.Title)
		}
		break
	}

	// Fall back to filename
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
