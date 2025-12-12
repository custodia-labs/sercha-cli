package outlook

import (
	"fmt"
	"strings"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Message represents an Outlook message from Microsoft Graph API.
type Message struct {
	ID                 string        `json:"id"`
	Subject            string        `json:"subject"`
	BodyPreview        string        `json:"bodyPreview"`
	Body               *MessageBody  `json:"body"`
	From               *EmailAddress `json:"from"`
	ToRecipients       []Recipient   `json:"toRecipients"`
	CcRecipients       []Recipient   `json:"ccRecipients"`
	ReceivedDateTime   string        `json:"receivedDateTime"`
	SentDateTime       string        `json:"sentDateTime"`
	IsRead             bool          `json:"isRead"`
	IsDraft            bool          `json:"isDraft"`
	Importance         string        `json:"importance"`
	ConversationID     string        `json:"conversationId"`
	ParentFolderID     string        `json:"parentFolderId"`
	WebLink            string        `json:"webLink"`
	HasAttachments     bool          `json:"hasAttachments"`
	InternetMessageID  string        `json:"internetMessageId"`
	ConversationIndex  string        `json:"conversationIndex"`
	InferenceClassific string        `json:"inferenceClassification"`
}

// MessageBody represents the body of an email.
type MessageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// EmailAddress represents an email address with optional name.
type EmailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// Recipient represents an email recipient.
type Recipient struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// DeltaResponse represents a delta query response from Microsoft Graph.
type DeltaResponse struct {
	Value         []Message `json:"value"`
	NextLink      string    `json:"@odata.nextLink"`
	DeltaLink     string    `json:"@odata.deltaLink"`
	DeltaRemoved  []Removed `json:"@removed,omitempty"`
	Context       string    `json:"@odata.context"`
	DeltaNextLink string    `json:"@delta.link,omitempty"`
}

// Removed represents a deleted item in delta response.
type Removed struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// MessageWithRemoved wraps a message that may have been deleted.
type MessageWithRemoved struct {
	Message
	Removed *struct {
		Reason string `json:"reason"`
	} `json:"@removed,omitempty"`
}

// MessageToRawDocument converts an Outlook message to a RawDocument.
func MessageToRawDocument(msg *Message, sourceID string) *domain.RawDocument {
	content := buildMessageContent(msg)
	parentURI := buildParentURI(msg)

	metadata := map[string]any{
		"message_id":      msg.ID,
		"subject":         msg.Subject,
		"conversation_id": msg.ConversationID,
		"folder_id":       msg.ParentFolderID,
		"is_read":         msg.IsRead,
		"is_draft":        msg.IsDraft,
		"importance":      msg.Importance,
		"has_attachments": msg.HasAttachments,
	}

	if msg.From != nil {
		metadata["from"] = msg.From.EmailAddress.Address
		metadata["from_name"] = msg.From.EmailAddress.Name
	}

	if msg.ReceivedDateTime != "" {
		metadata["received_at"] = msg.ReceivedDateTime
	}
	if msg.SentDateTime != "" {
		metadata["sent_at"] = msg.SentDateTime
	}
	if msg.WebLink != "" {
		metadata["web_link"] = msg.WebLink
	}
	if msg.InternetMessageID != "" {
		metadata["internet_message_id"] = msg.InternetMessageID
	}

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("outlook://messages/%s", msg.ID),
		MIMEType:  "message/rfc822",
		Content:   []byte(content),
		ParentURI: parentURI,
		Metadata:  metadata,
	}
}

// buildMessageContent builds the text content of the message.
func buildMessageContent(msg *Message) string {
	var sb strings.Builder

	// Subject
	if msg.Subject != "" {
		sb.WriteString("Subject: ")
		sb.WriteString(msg.Subject)
		sb.WriteString("\n")
	}

	// From
	if msg.From != nil {
		sb.WriteString("From: ")
		if msg.From.EmailAddress.Name != "" {
			sb.WriteString(msg.From.EmailAddress.Name)
			sb.WriteString(" <")
			sb.WriteString(msg.From.EmailAddress.Address)
			sb.WriteString(">")
		} else {
			sb.WriteString(msg.From.EmailAddress.Address)
		}
		sb.WriteString("\n")
	}

	// To
	if len(msg.ToRecipients) > 0 {
		sb.WriteString("To: ")
		sb.WriteString(formatRecipients(msg.ToRecipients))
		sb.WriteString("\n")
	}

	// Cc
	if len(msg.CcRecipients) > 0 {
		sb.WriteString("Cc: ")
		sb.WriteString(formatRecipients(msg.CcRecipients))
		sb.WriteString("\n")
	}

	// Date
	if msg.ReceivedDateTime != "" {
		if t, err := time.Parse(time.RFC3339, msg.ReceivedDateTime); err == nil {
			sb.WriteString("Date: ")
			sb.WriteString(t.Format(time.RFC1123Z))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	// Body
	if msg.Body != nil && msg.Body.Content != "" {
		sb.WriteString(msg.Body.Content)
	} else if msg.BodyPreview != "" {
		sb.WriteString(msg.BodyPreview)
	}

	return sb.String()
}

// formatRecipients formats a list of recipients for display.
func formatRecipients(recipients []Recipient) string {
	names := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if r.EmailAddress.Name != "" {
			names = append(names, fmt.Sprintf("%s <%s>",
				r.EmailAddress.Name, r.EmailAddress.Address))
		} else if r.EmailAddress.Address != "" {
			names = append(names, r.EmailAddress.Address)
		}
	}
	return strings.Join(names, ", ")
}

// buildParentURI builds a parent URI for conversation threading.
func buildParentURI(msg *Message) *string {
	if msg.ConversationID != "" && msg.ConversationID != msg.ID {
		uri := fmt.Sprintf("outlook://conversations/%s", msg.ConversationID)
		return &uri
	}
	return nil
}

// ShouldSyncMessage checks if a message should be synced based on config.
func ShouldSyncMessage(msg *Message, cfg *Config) bool {
	if msg == nil || msg.ID == "" {
		return false
	}
	return true
}

// IsMessageRemoved checks if the message was marked as removed in delta response.
func IsMessageRemoved(msg *MessageWithRemoved) bool {
	return msg.Removed != nil
}
