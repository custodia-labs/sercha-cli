package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestExclusion_Fields tests Exclusion structure fields
func TestExclusion_Fields(t *testing.T) {
	now := time.Now()
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///excluded/document.pdf",
		Reason:     "User requested exclusion",
		ExcludedAt: now,
	}

	assert.Equal(t, "excl-123", exclusion.ID)
	assert.Equal(t, "source-456", exclusion.SourceID)
	assert.Equal(t, "doc-789", exclusion.DocumentID)
	assert.Equal(t, "file:///excluded/document.pdf", exclusion.URI)
	assert.Equal(t, "User requested exclusion", exclusion.Reason)
	assert.Equal(t, now, exclusion.ExcludedAt)
}

// TestExclusion_EmptyReason tests exclusion without a reason
func TestExclusion_EmptyReason(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///excluded.txt",
		Reason:     "",
		ExcludedAt: time.Now(),
	}

	assert.Empty(t, exclusion.Reason)
}

// TestExclusion_WithReason tests exclusion with various reasons
func TestExclusion_WithReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"user requested", "User requested exclusion"},
		{"duplicate", "Duplicate content"},
		{"irrelevant", "Content not relevant to search"},
		{"private", "Contains private information"},
		{"malformed", "Document is malformed or corrupted"},
		{"test data", "Test data, not production"},
		{"long reason", "This is a very long reason that explains in great detail why this document was excluded from the search index. It contains multiple sentences and provides comprehensive context."},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exclusion := Exclusion{
				ID:         "excl-123",
				SourceID:   "source-456",
				DocumentID: "doc-789",
				URI:        "file:///test.txt",
				Reason:     tt.reason,
				ExcludedAt: time.Now(),
			}
			assert.Equal(t, tt.reason, exclusion.Reason)
		})
	}
}

// TestExclusion_TimeFields tests exclusion timestamp
func TestExclusion_TimeFields(t *testing.T) {
	past := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Test exclusion",
		ExcludedAt: past,
	}

	assert.Equal(t, past, exclusion.ExcludedAt)
	assert.True(t, time.Since(exclusion.ExcludedAt) > 0)
}

// TestExclusion_ZeroTime tests exclusion with zero time
func TestExclusion_ZeroTime(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Test exclusion",
		ExcludedAt: time.Time{},
	}

	assert.True(t, exclusion.ExcludedAt.IsZero())
}

// TestExclusion_RecentExclusion tests recently excluded document
func TestExclusion_RecentExclusion(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Recently excluded",
		ExcludedAt: time.Now().Add(-5 * time.Minute),
	}

	timeSince := time.Since(exclusion.ExcludedAt)
	assert.True(t, timeSince < 10*time.Minute)
	assert.True(t, timeSince > 0)
}

// TestExclusion_OldExclusion tests old excluded document
func TestExclusion_OldExclusion(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Excluded long ago",
		ExcludedAt: time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
	}

	timeSince := time.Since(exclusion.ExcludedAt)
	assert.True(t, timeSince > 364*24*time.Hour)
}

// TestExclusion_MultipleFromSource tests multiple exclusions from same source
func TestExclusion_MultipleFromSource(t *testing.T) {
	sourceID := "source-456"
	exclusions := []Exclusion{
		{
			ID:         "excl-1",
			SourceID:   sourceID,
			DocumentID: "doc-1",
			URI:        "file:///doc1.txt",
			Reason:     "Reason 1",
			ExcludedAt: time.Now(),
		},
		{
			ID:         "excl-2",
			SourceID:   sourceID,
			DocumentID: "doc-2",
			URI:        "file:///doc2.txt",
			Reason:     "Reason 2",
			ExcludedAt: time.Now(),
		},
		{
			ID:         "excl-3",
			SourceID:   sourceID,
			DocumentID: "doc-3",
			URI:        "file:///doc3.txt",
			Reason:     "Reason 3",
			ExcludedAt: time.Now(),
		},
	}

	for _, excl := range exclusions {
		assert.Equal(t, sourceID, excl.SourceID)
	}

	// Verify all have unique IDs and document IDs
	assert.NotEqual(t, exclusions[0].ID, exclusions[1].ID)
	assert.NotEqual(t, exclusions[0].DocumentID, exclusions[1].DocumentID)
}

// TestExclusion_URIFormats tests various URI formats
func TestExclusion_URIFormats(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"file path", "file:///path/to/excluded.pdf"},
		{"http url", "https://example.com/excluded"},
		{"drive url", "drive://file-id-excluded"},
		{"email", "gmail://message-id-excluded"},
		{"relative path", "documents/excluded.pdf"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exclusion := Exclusion{
				ID:         "excl-123",
				SourceID:   "source-456",
				DocumentID: "doc-789",
				URI:        tt.uri,
				Reason:     "Test",
				ExcludedAt: time.Now(),
			}
			assert.Equal(t, tt.uri, exclusion.URI)
		})
	}
}

// TestExclusion_EmptyStrings tests exclusion with empty string fields
func TestExclusion_EmptyStrings(t *testing.T) {
	exclusion := Exclusion{
		ID:         "",
		SourceID:   "",
		DocumentID: "",
		URI:        "",
		Reason:     "",
		ExcludedAt: time.Now(),
	}

	assert.Empty(t, exclusion.ID)
	assert.Empty(t, exclusion.SourceID)
	assert.Empty(t, exclusion.DocumentID)
	assert.Empty(t, exclusion.URI)
	assert.Empty(t, exclusion.Reason)
}

// TestExclusion_SpecialCharacters tests exclusion with special characters
func TestExclusion_SpecialCharacters(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///path/with spaces/and@special#chars.pdf",
		Reason:     "Reason with émojis 🚫 and unicode: 文档",
		ExcludedAt: time.Now(),
	}

	assert.Contains(t, exclusion.URI, " ")
	assert.Contains(t, exclusion.URI, "@")
	assert.Contains(t, exclusion.URI, "#")
	assert.Contains(t, exclusion.Reason, "🚫")
	assert.Contains(t, exclusion.Reason, "文档")
}

// TestExclusion_ReasonLength tests exclusions with various reason lengths
func TestExclusion_ReasonLength(t *testing.T) {
	tests := []struct {
		name      string
		reasonLen int
	}{
		{"short", 10},
		{"medium", 100},
		{"long", 500},
		{"very long", 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := string(make([]byte, tt.reasonLen))
			exclusion := Exclusion{
				ID:         "excl-123",
				SourceID:   "source-456",
				DocumentID: "doc-789",
				URI:        "file:///test.txt",
				Reason:     reason,
				ExcludedAt: time.Now(),
			}
			assert.Len(t, exclusion.Reason, tt.reasonLen)
		})
	}
}

// TestExclusion_DocumentRelationship tests relationship to document
func TestExclusion_DocumentRelationship(t *testing.T) {
	docID := "doc-789"
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: docID,
		URI:        "file:///test.txt",
		Reason:     "Test",
		ExcludedAt: time.Now(),
	}

	// Exclusion references the document
	assert.Equal(t, docID, exclusion.DocumentID)
}

// TestExclusion_SourceRelationship tests relationship to source
func TestExclusion_SourceRelationship(t *testing.T) {
	sourceID := "source-456"
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   sourceID,
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Test",
		ExcludedAt: time.Now(),
	}

	// Exclusion references the source
	assert.Equal(t, sourceID, exclusion.SourceID)
}

// TestExclusion_FutureExclusion tests exclusion with future timestamp (edge case)
func TestExclusion_FutureExclusion(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		Reason:     "Future exclusion",
		ExcludedAt: future,
	}

	assert.True(t, exclusion.ExcludedAt.After(time.Now()))
}

// TestExclusion_SameDocumentMultipleSources tests same document excluded from different sources
func TestExclusion_SameDocumentMultipleSources(t *testing.T) {
	docID := "doc-789"
	uri := "file:///shared.txt"

	exclusions := []Exclusion{
		{
			ID:         "excl-1",
			SourceID:   "source-1",
			DocumentID: docID,
			URI:        uri,
			Reason:     "Excluded from source 1",
			ExcludedAt: time.Now(),
		},
		{
			ID:         "excl-2",
			SourceID:   "source-2",
			DocumentID: docID,
			URI:        uri,
			Reason:     "Excluded from source 2",
			ExcludedAt: time.Now(),
		},
	}

	// Same document and URI
	assert.Equal(t, exclusions[0].DocumentID, exclusions[1].DocumentID)
	assert.Equal(t, exclusions[0].URI, exclusions[1].URI)

	// Different sources and IDs
	assert.NotEqual(t, exclusions[0].SourceID, exclusions[1].SourceID)
	assert.NotEqual(t, exclusions[0].ID, exclusions[1].ID)
}

// TestExclusion_MinimalFields tests exclusion with only required fields
func TestExclusion_MinimalFields(t *testing.T) {
	exclusion := Exclusion{
		ID:         "excl-123",
		SourceID:   "source-456",
		DocumentID: "doc-789",
		URI:        "file:///test.txt",
		ExcludedAt: time.Now(),
		// Reason is optional
	}

	assert.NotEmpty(t, exclusion.ID)
	assert.NotEmpty(t, exclusion.SourceID)
	assert.NotEmpty(t, exclusion.DocumentID)
	assert.NotEmpty(t, exclusion.URI)
	assert.Empty(t, exclusion.Reason)
}
