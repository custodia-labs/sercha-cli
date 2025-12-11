package eml

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNew(t *testing.T) {
	normaliser := New()
	require.NotNil(t, normaliser)
	assert.IsType(t, &Normaliser{}, normaliser)
}

func TestSupportedMIMETypes(t *testing.T) {
	normaliser := New()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, "message/rfc822")
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

func TestNormalise_SimpleEmail(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	emlContent := `From: sender@example.com
To: recipient@example.com
Subject: Test Email Subject
Date: Mon, 01 Jan 2024 10:00:00 +0000
Content-Type: text/plain

This is the body of the email.
It has multiple lines.
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, "test-source", doc.SourceID)
	assert.Equal(t, "/path/to/email.eml", doc.URI)
	assert.Equal(t, "Test Email Subject", doc.Title)
	assert.Contains(t, doc.Content, "This is the body of the email")
	assert.Contains(t, doc.Content, "sender@example.com")
	assert.Contains(t, doc.Content, "recipient@example.com")
	assert.Equal(t, "message/rfc822", doc.Metadata["mime_type"])
	assert.Equal(t, "eml", doc.Metadata["format"])
	assert.Equal(t, "sender@example.com", doc.Metadata["from"])
	assert.Equal(t, "recipient@example.com", doc.Metadata["to"])
}

func TestNormalise_NoSubject(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	emlContent := `From: sender@example.com
To: recipient@example.com
Content-Type: text/plain

Email without subject.
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/my_email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should fall back to filename as title
	assert.Equal(t, "my email", result.Document.Title)
}

func TestNormalise_HTMLBody(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	emlContent := `From: sender@example.com
To: recipient@example.com
Subject: HTML Email
Content-Type: text/html

<html>
<body>
<h1>Hello</h1>
<p>This is <b>HTML</b> content.</p>
</body>
</html>
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// HTML tags should be stripped
	assert.Contains(t, result.Document.Content, "Hello")
	assert.Contains(t, result.Document.Content, "HTML content")
	assert.NotContains(t, result.Document.Content, "<h1>")
	assert.NotContains(t, result.Document.Content, "<p>")
}

func TestNormalise_MultipartAlternative(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	emlContent := `From: sender@example.com
To: recipient@example.com
Subject: Multipart Email
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain

Plain text version of the email.
--boundary123
Content-Type: text/html

<html><body><p>HTML version</p></body></html>
--boundary123--
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should prefer plain text over HTML
	assert.Contains(t, result.Document.Content, "Plain text version")
}

func TestNormalise_EncodedSubject(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	// RFC 2047 encoded subject (UTF-8 base64)
	emlContent := `From: sender@example.com
To: recipient@example.com
Subject: =?UTF-8?B?VGVzdCBFbWFpbCDwn5iA?=
Content-Type: text/plain

Body content.
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should decode the subject
	assert.NotEmpty(t, result.Document.Title)
}

func TestNormalise_InvalidEmail(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte("not a valid email"),
	}

	result, err := normaliser.Normalise(ctx, raw)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestNormalise_WithMetadata(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	emlContent := `From: sender@example.com
To: recipient@example.com
Subject: Test
Content-Type: text/plain

Body.
`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/email.eml",
		MIMEType: "message/rfc822",
		Content:  []byte(emlContent),
		Metadata: map[string]any{
			"folder": "inbox",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should preserve original metadata
	assert.Equal(t, "inbox", result.Document.Metadata["folder"])
	// And add email-specific metadata
	assert.Equal(t, "eml", result.Document.Metadata["format"])
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Simple Subject",
			expected: "Simple Subject",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "utf8 base64 encoded",
			input:    "=?UTF-8?B?SGVsbG8gV29ybGQ=?=",
			expected: "Hello World",
		},
		{
			name:     "utf8 quoted printable",
			input:    "=?UTF-8?Q?Hello_World?=",
			expected: "Hello World",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := decodeHeader(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple html",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "nested tags",
			input:    "<div><p>Hello <b>World</b></p></div>",
			expected: "Hello World",
		},
		{
			name:     "with whitespace",
			input:    "<p>Line 1</p>\n\n<p>Line 2</p>",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "no tags",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripHTMLTags(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractTitleFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "simple filename",
			uri:      "/path/to/email.eml",
			expected: "email",
		},
		{
			name:     "with underscores",
			uri:      "/path/to/my_email_file.eml",
			expected: "my email file",
		},
		{
			name:     "with dashes",
			uri:      "/path/to/my-email-file.eml",
			expected: "my email file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTitleFromURI(tc.uri)
			assert.Equal(t, tc.expected, result)
		})
	}
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

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}
