package pdf

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockRunner is a test double for CommandRunner.
type mockRunner struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return m.output, m.err
}

func TestNew(t *testing.T) {
	normaliser := New()
	require.NotNil(t, normaliser)
	assert.IsType(t, &Normaliser{}, normaliser)
}

func TestSupportedMIMETypes(t *testing.T) {
	normaliser := New()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, "application/pdf")
	assert.Len(t, mimeTypes, 1)
}

func TestSupportedConnectorTypes(t *testing.T) {
	normaliser := New()
	assert.Nil(t, normaliser.SupportedConnectorTypes())
}

func TestPriority(t *testing.T) {
	normaliser := New()
	assert.Equal(t, 50, normaliser.Priority())
}

func TestNormalise_NilDocument(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		uri      string
		expected string
	}{
		{
			name:     "first line as title",
			content:  "Document Title\n\nSome content here.",
			uri:      "/doc.pdf",
			expected: "Document Title",
		},
		{
			name:     "skip empty lines",
			content:  "\n\n\nActual Title\nContent",
			uri:      "/doc.pdf",
			expected: "Actual Title",
		},
		{
			name:     "fallback to filename",
			content:  "",
			uri:      "/path/to/my_document.pdf",
			expected: "my document",
		},
		{
			name:     "skip very long first line",
			content:  string(make([]byte, 250)) + "\nShort Title\nContent",
			uri:      "/doc.pdf",
			expected: "Short Title",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTitle(tc.content, tc.uri)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInstallInstructions(t *testing.T) {
	instructions := InstallInstructions()
	assert.Contains(t, instructions, "pdftotext")
	assert.Contains(t, instructions, "brew install poppler")
	assert.Contains(t, instructions, "apt install poppler-utils")
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}

func TestCopyMetadata(t *testing.T) {
	tests := []struct {
		name string
		src  map[string]any
	}{
		{
			name: "nil map",
			src:  nil,
		},
		{
			name: "empty map",
			src:  map[string]any{},
		},
		{
			name: "with values",
			src: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := copyMetadata(tc.src)
			if tc.src == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, len(tc.src), len(result))
				for k, v := range tc.src {
					assert.Equal(t, v, result[k])
				}
			}
		})
	}
}

// Integration test - only runs if pdftotext is available.
func TestNormalise_Integration(t *testing.T) {
	if err := CheckAvailable(); err != nil {
		t.Skip("pdftotext not available, skipping integration test")
	}

	// This test would require a real PDF file.
	// For CI, we rely on the mock tests above.
	t.Skip("integration test requires sample PDF file")
}

func TestErrPDFToolNotFound(t *testing.T) {
	assert.Error(t, ErrPDFToolNotFound)
	assert.Contains(t, ErrPDFToolNotFound.Error(), "pdftotext")
}

// TestNewWithRunner verifies the mock runner injection works.
func TestNewWithRunner(t *testing.T) {
	runner := &mockRunner{output: []byte("test output"), err: nil}
	normaliser := NewWithRunner(runner)
	require.NotNil(t, normaliser)
	assert.Equal(t, runner, normaliser.runner)
}

// TestNormalise_WithMockRunner tests normalisation with a mocked pdftotext.
func TestNormalise_WithMockRunner(t *testing.T) {
	// Skip if pdftotext not in PATH (LookPath check happens before runner).
	if err := CheckAvailable(); err != nil {
		t.Skip("pdftotext not in PATH, skipping mock runner test")
	}

	runner := &mockRunner{
		output: []byte("PDF Title\n\nThis is the content of the PDF.\n"),
		err:    nil,
	}
	normaliser := NewWithRunner(runner)
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.pdf",
		MIMEType: "application/pdf",
		Content:  []byte("%PDF-1.4 fake pdf content"),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, "test-source", doc.SourceID)
	assert.Equal(t, "/path/to/document.pdf", doc.URI)
	assert.Equal(t, "PDF Title", doc.Title)
	assert.Contains(t, doc.Content, "This is the content of the PDF.")
	assert.Equal(t, "application/pdf", doc.Metadata["mime_type"])
	assert.Equal(t, "pdf", doc.Metadata["format"])
}

// TestNormalise_RunnerError tests error handling when pdftotext fails.
func TestNormalise_RunnerError(t *testing.T) {
	if err := CheckAvailable(); err != nil {
		t.Skip("pdftotext not in PATH, skipping runner error test")
	}

	runner := &mockRunner{
		output: nil,
		err:    errors.New("pdftotext crashed"),
	}
	normaliser := NewWithRunner(runner)
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/document.pdf",
		MIMEType: "application/pdf",
		Content:  []byte("%PDF-1.4 fake pdf content"),
	}

	result, err := normaliser.Normalise(ctx, raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pdftotext failed")
	assert.Nil(t, result)
}
