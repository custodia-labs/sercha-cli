package outlook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageToRawDocument(t *testing.T) {
	msg := &Message{
		ID:      "AAMkAGI2ABC123",
		Subject: "Test Subject",
		Body: &MessageBody{
			ContentType: "text",
			Content:     "This is the email body content.",
		},
		From: &EmailAddress{
			EmailAddress: struct {
				Name    string `json:"name"`
				Address string `json:"address"`
			}{
				Name:    "John Doe",
				Address: "john@example.com",
			},
		},
		ToRecipients: []Recipient{
			{
				EmailAddress: struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				}{
					Name:    "Jane Smith",
					Address: "jane@example.com",
				},
			},
		},
		ReceivedDateTime: "2024-01-15T10:30:00Z",
		SentDateTime:     "2024-01-15T10:29:00Z",
		IsRead:           true,
		IsDraft:          false,
		Importance:       "normal",
		ConversationID:   "AAQkAGI2CONV123",
		ParentFolderID:   "inbox",
		WebLink:          "https://outlook.office.com/mail/id/AAMkAGI2ABC123",
		HasAttachments:   false,
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.Equal(t, "source-abc", doc.SourceID)
	assert.Equal(t, "outlook://messages/AAMkAGI2ABC123", doc.URI)
	assert.Equal(t, "message/rfc822", doc.MIMEType)

	// Check content includes relevant fields
	content := string(doc.Content)
	assert.Contains(t, content, "Subject: Test Subject")
	assert.Contains(t, content, "From: John Doe <john@example.com>")
	assert.Contains(t, content, "To: Jane Smith <jane@example.com>")
	assert.Contains(t, content, "This is the email body content.")

	// Check metadata
	assert.Equal(t, "AAMkAGI2ABC123", doc.Metadata["message_id"])
	assert.Equal(t, "Test Subject", doc.Metadata["subject"])
	assert.Equal(t, "AAQkAGI2CONV123", doc.Metadata["conversation_id"])
	assert.Equal(t, "inbox", doc.Metadata["folder_id"])
	assert.Equal(t, true, doc.Metadata["is_read"])
	assert.Equal(t, false, doc.Metadata["is_draft"])
	assert.Equal(t, "normal", doc.Metadata["importance"])
	assert.Equal(t, "john@example.com", doc.Metadata["from"])
	assert.Equal(t, "John Doe", doc.Metadata["from_name"])
	assert.Equal(t, "2024-01-15T10:30:00Z", doc.Metadata["received_at"])
}

func TestMessageToRawDocument_WithParentURI(t *testing.T) {
	msg := &Message{
		ID:             "AAMkAGI2MSG123",
		Subject:        "Reply: Test",
		ConversationID: "AAQkAGI2CONV999", // Different from message ID
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "outlook://conversations/AAQkAGI2CONV999", *doc.ParentURI)
}

func TestMessageToRawDocument_NoParentURI(t *testing.T) {
	msg := &Message{
		ID:             "AAMkAGI2MSG123",
		Subject:        "New Thread",
		ConversationID: "", // Empty conversation ID
	}

	doc := MessageToRawDocument(msg, "source-abc")

	assert.Nil(t, doc.ParentURI)
}

func TestBuildMessageContent(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		contains []string
		excludes []string
	}{
		{
			name: "full message",
			msg: &Message{
				Subject: "Meeting Tomorrow",
				From: &EmailAddress{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Name:    "Alice",
						Address: "alice@example.com",
					},
				},
				ToRecipients: []Recipient{
					{
						EmailAddress: struct {
							Name    string `json:"name"`
							Address string `json:"address"`
						}{
							Name:    "Bob",
							Address: "bob@example.com",
						},
					},
				},
				Body: &MessageBody{
					Content: "Let's meet at 3pm.",
				},
			},
			contains: []string{
				"Subject: Meeting Tomorrow",
				"From: Alice <alice@example.com>",
				"To: Bob <bob@example.com>",
				"Let's meet at 3pm.",
			},
		},
		{
			name: "message with body preview",
			msg: &Message{
				Subject:     "Quick Note",
				BodyPreview: "This is a preview...",
			},
			contains: []string{"Quick Note", "This is a preview..."},
		},
		{
			name: "message with multiple recipients",
			msg: &Message{
				ToRecipients: []Recipient{
					{
						EmailAddress: struct {
							Name    string `json:"name"`
							Address string `json:"address"`
						}{
							Name:    "Alice",
							Address: "alice@example.com",
						},
					},
					{
						EmailAddress: struct {
							Name    string `json:"name"`
							Address string `json:"address"`
						}{
							Address: "bob@example.com",
						},
					},
				},
			},
			contains: []string{"Alice <alice@example.com>", "bob@example.com"},
		},
		{
			name: "message with CC",
			msg: &Message{
				CcRecipients: []Recipient{
					{
						EmailAddress: struct {
							Name    string `json:"name"`
							Address string `json:"address"`
						}{
							Name:    "Manager",
							Address: "manager@example.com",
						},
					},
				},
			},
			contains: []string{"Cc: Manager <manager@example.com>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildMessageContent(tt.msg)

			for _, s := range tt.contains {
				assert.Contains(t, content, s)
			}
			for _, s := range tt.excludes {
				assert.NotContains(t, content, s)
			}
		})
	}
}

func TestFormatRecipients(t *testing.T) {
	tests := []struct {
		name       string
		recipients []Recipient
		expected   string
	}{
		{
			name:       "empty recipients",
			recipients: []Recipient{},
			expected:   "",
		},
		{
			name: "single recipient with name",
			recipients: []Recipient{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Name:    "Alice",
						Address: "alice@example.com",
					},
				},
			},
			expected: "Alice <alice@example.com>",
		},
		{
			name: "single recipient without name",
			recipients: []Recipient{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Address: "alice@example.com",
					},
				},
			},
			expected: "alice@example.com",
		},
		{
			name: "multiple recipients",
			recipients: []Recipient{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Name:    "Alice",
						Address: "alice@example.com",
					},
				},
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Address: "bob@example.com",
					},
				},
			},
			expected: "Alice <alice@example.com>, bob@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRecipients(tt.recipients)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSyncMessage(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		msg      *Message
		expected bool
	}{
		{
			name:     "valid message",
			msg:      &Message{ID: "msg-123"},
			expected: true,
		},
		{
			name:     "nil message",
			msg:      nil,
			expected: false,
		},
		{
			name:     "empty message ID",
			msg:      &Message{ID: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSyncMessage(tt.msg, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMessageRemoved(t *testing.T) {
	tests := []struct {
		name     string
		msg      *MessageWithRemoved
		expected bool
	}{
		{
			name: "not removed",
			msg: &MessageWithRemoved{
				Message: Message{ID: "msg-123"},
				Removed: nil,
			},
			expected: false,
		},
		{
			name: "removed",
			msg: &MessageWithRemoved{
				Message: Message{ID: "msg-123"},
				Removed: &struct {
					Reason string `json:"reason"`
				}{Reason: "deleted"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMessageRemoved(tt.msg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildParentURI(t *testing.T) {
	tests := []struct {
		name      string
		msg       *Message
		expectNil bool
		expected  string
	}{
		{
			name: "has conversation ID different from message ID",
			msg: &Message{
				ID:             "msg-123",
				ConversationID: "conv-456",
			},
			expectNil: false,
			expected:  "outlook://conversations/conv-456",
		},
		{
			name: "no conversation ID",
			msg: &Message{
				ID:             "msg-123",
				ConversationID: "",
			},
			expectNil: true,
		},
		{
			name: "conversation ID same as message ID",
			msg: &Message{
				ID:             "msg-123",
				ConversationID: "msg-123",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildParentURI(tt.msg)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}
