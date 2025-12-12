package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// ErrPDFToolNotFound is returned when pdftotext is not installed.
var ErrPDFToolNotFound = errors.New("pdftotext not found: install poppler-utils")

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// CommandRunner abstracts command execution for testing.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// DefaultRunner executes commands using os/exec.
type DefaultRunner struct{}

// Run executes a command and returns its output.
func (r *DefaultRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Include stderr in error for better debugging
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

// Normaliser handles PDF documents using pdftotext.
type Normaliser struct {
	runner CommandRunner
}

// New creates a new PDF normaliser.
func New() *Normaliser {
	return &Normaliser{
		runner: &DefaultRunner{},
	}
}

// NewWithRunner creates a PDF normaliser with a custom command runner (for testing).
func NewWithRunner(runner CommandRunner) *Normaliser {
	return &Normaliser{
		runner: runner,
	}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{"application/pdf"}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 50 // Generic MIME normaliser
}

// Normalise converts a PDF document to a normalised document.
func (n *Normaliser) Normalise(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Check if pdftotext is available
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return nil, ErrPDFToolNotFound
	}

	// Write content to temp file (pdftotext needs a file)
	tmpFile, err := os.CreateTemp("", "sercha-pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(raw.Content); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Run pdftotext: -layout preserves layout, - outputs to stdout
	output, err := n.runner.Run(ctx, "pdftotext", "-layout", tmpFile.Name(), "-")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Incorrect password") ||
				strings.Contains(stderr, "encrypted") {
				return nil, fmt.Errorf("PDF is password-protected")
			}
		}
		return nil, fmt.Errorf("pdftotext failed: %w", err)
	}

	content := strings.TrimSpace(string(output))

	// Extract title: check metadata first, then PDF content, then filename
	title := extractTitleFromMetadataOrContent(raw, content)

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
	doc.Metadata["format"] = "pdf"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// extractTitleFromMetadataOrContent checks metadata for title first, then falls back to content/URI.
// This supports connectors like Google Drive that set Metadata["title"] to the actual file name.
func extractTitleFromMetadataOrContent(raw *domain.RawDocument, content string) string {
	if raw.Metadata != nil {
		if title, ok := raw.Metadata["title"].(string); ok && title != "" {
			return title
		}
	}
	return extractTitle(content, raw.URI)
}

// extractTitle gets the title from PDF content or falls back to filename.
func extractTitle(content, uri string) string {
	// Try first non-empty line as title
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) < 200 { // Reasonable title length
			return line
		}
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

// CheckAvailable returns nil if pdftotext is installed, error otherwise.
func CheckAvailable() error {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return ErrPDFToolNotFound
	}
	return nil
}

// InstallInstructions returns platform-specific install instructions.
func InstallInstructions() string {
	return bytes.NewBufferString(`pdftotext is required for PDF support.

Install poppler-utils:
  macOS:   brew install poppler
  Ubuntu:  sudo apt install poppler-utils
  Fedora:  sudo dnf install poppler-utils
`).String()
}
