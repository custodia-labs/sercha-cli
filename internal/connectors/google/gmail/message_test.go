package gmail

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestMessageToRawDocument(t *testing.T) {
	// Create a test message with base64url-encoded content
	rawContent := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nHello!"
	encodedContent := base64.URLEncoding.EncodeToString([]byte(rawContent))

	msg := &gmail.Message{
		Id:           "msg-123",
		ThreadId:     "thread-456",
		Raw:          encodedContent,
		LabelIds:     []string{"INBOX", "UNREAD"},
		Snippet:      "Hello!",
		HistoryId:    789,
		InternalDate: 1234567890000,
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.Equal(t, "source-abc", doc.SourceID)
	assert.Equal(t, "gmail://messages/msg-123", doc.URI)
	assert.Equal(t, "message/rfc822", doc.MIMEType)
	assert.Equal(t, []byte(rawContent), doc.Content)

	// Check metadata
	assert.Equal(t, "msg-123", doc.Metadata["message_id"])
	assert.Equal(t, "thread-456", doc.Metadata["thread_id"])
	assert.Equal(t, []string{"INBOX", "UNREAD"}, doc.Metadata["labels"])
	assert.Equal(t, "Hello!", doc.Metadata["snippet"])
	assert.Equal(t, uint64(789), doc.Metadata["history_id"])
	assert.Equal(t, int64(1234567890000), doc.Metadata["internal_date"])
}

func TestMessageToRawDocument_WithThreadParent(t *testing.T) {
	msg := &gmail.Message{
		Id:       "msg-123",
		ThreadId: "thread-456", // Different from message ID
		Raw:      base64.URLEncoding.EncodeToString([]byte("test")),
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "gmail://threads/thread-456", *doc.ParentURI)
}

func TestMessageToRawDocument_NoThreadParent(t *testing.T) {
	// When thread ID equals message ID, no parent
	msg := &gmail.Message{
		Id:       "msg-123",
		ThreadId: "msg-123", // Same as message ID
		Raw:      base64.URLEncoding.EncodeToString([]byte("test")),
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.Nil(t, doc.ParentURI)
}

func TestMessageToRawDocument_InvalidBase64(t *testing.T) {
	msg := &gmail.Message{
		Id:       "msg-123",
		ThreadId: "thread-456",
		Raw:      "not-valid-base64!!!",
	}

	doc := MessageToRawDocument(msg, "source-abc")

	// Should not panic, should return empty content
	assert.Empty(t, doc.Content)
}

func TestShouldSyncMessage(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		config   *Config
		expected bool
	}{
		{
			name:   "matches required label",
			labels: []string{"INBOX", "UNREAD"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: false,
			},
			expected: true,
		},
		{
			name:   "no required labels configured - sync all",
			labels: []string{"SENT"},
			config: &Config{
				LabelIDs:         []string{},
				IncludeSpamTrash: false,
			},
			expected: true,
		},
		{
			name:   "does not match required label",
			labels: []string{"SENT", "IMPORTANT"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: false,
			},
			expected: false,
		},
		{
			name:   "spam message excluded by default",
			labels: []string{"INBOX", "SPAM"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: false,
			},
			expected: false,
		},
		{
			name:   "trash message excluded by default",
			labels: []string{"INBOX", "TRASH"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: false,
			},
			expected: false,
		},
		{
			name:   "spam included when configured",
			labels: []string{"INBOX", "SPAM"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: true,
			},
			expected: true,
		},
		{
			name:   "trash included when configured",
			labels: []string{"INBOX", "TRASH"},
			config: &Config{
				LabelIDs:         []string{"INBOX"},
				IncludeSpamTrash: true,
			},
			expected: true,
		},
		{
			name:   "matches one of multiple required labels",
			labels: []string{"STARRED"},
			config: &Config{
				LabelIDs:         []string{"INBOX", "STARRED", "IMPORTANT"},
				IncludeSpamTrash: false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &gmail.Message{
				LabelIds: tt.labels,
			}

			result := ShouldSyncMessage(msg, tt.config)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasRequiredLabel(t *testing.T) {
	tests := []struct {
		name           string
		msgLabels      []string
		requiredLabels []string
		expected       bool
	}{
		{
			name:           "empty required labels matches all",
			msgLabels:      []string{"INBOX"},
			requiredLabels: []string{},
			expected:       true,
		},
		{
			name:           "exact match",
			msgLabels:      []string{"INBOX"},
			requiredLabels: []string{"INBOX"},
			expected:       true,
		},
		{
			name:           "one of multiple matches",
			msgLabels:      []string{"INBOX", "STARRED"},
			requiredLabels: []string{"STARRED"},
			expected:       true,
		},
		{
			name:           "no match",
			msgLabels:      []string{"SENT"},
			requiredLabels: []string{"INBOX"},
			expected:       false,
		},
		{
			name:           "empty message labels",
			msgLabels:      []string{},
			requiredLabels: []string{"INBOX"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRequiredLabel(tt.msgLabels, tt.requiredLabels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSpamOrTrash(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected bool
	}{
		{
			name:     "has SPAM",
			labels:   []string{"INBOX", "SPAM"},
			expected: true,
		},
		{
			name:     "has TRASH",
			labels:   []string{"INBOX", "TRASH"},
			expected: true,
		},
		{
			name:     "no spam or trash",
			labels:   []string{"INBOX", "UNREAD"},
			expected: false,
		},
		{
			name:     "empty labels",
			labels:   []string{},
			expected: false,
		},
		{
			name:     "only SPAM",
			labels:   []string{"SPAM"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSpamOrTrash(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildParentURI(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		threadID  string
		expectNil bool
		expected  string
	}{
		{
			name:      "different thread ID creates parent",
			messageID: "msg-123",
			threadID:  "thread-456",
			expectNil: false,
			expected:  "gmail://threads/thread-456",
		},
		{
			name:      "same thread ID as message - no parent",
			messageID: "msg-123",
			threadID:  "msg-123",
			expectNil: true,
		},
		{
			name:      "empty thread ID - no parent",
			messageID: "msg-123",
			threadID:  "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &gmail.Message{
				Id:       tt.messageID,
				ThreadId: tt.threadID,
			}

			result := buildParentURI(msg)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}
