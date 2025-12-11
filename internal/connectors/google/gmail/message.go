package gmail

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/api/gmail/v1"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MessageToRawDocument converts a Gmail message to a RawDocument.
// When using Format("raw"), msg.Raw contains the base64url-encoded RFC 2822 message.
func MessageToRawDocument(msg *gmail.Message, sourceID string) *domain.RawDocument {
	// Decode the base64url-encoded raw message (RFC 2822 format)
	rawBytes, err := base64.URLEncoding.DecodeString(msg.Raw)
	if err != nil {
		// Fall back to empty content if decoding fails
		rawBytes = []byte{}
	}
	parentURI := buildParentURI(msg)

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("gmail://messages/%s", msg.Id),
		MIMEType:  "message/rfc822",
		Content:   rawBytes,
		ParentURI: parentURI,
		Metadata: map[string]any{
			"message_id":    msg.Id,
			"thread_id":     msg.ThreadId,
			"labels":        msg.LabelIds,
			"snippet":       msg.Snippet,
			"history_id":    msg.HistoryId,
			"internal_date": msg.InternalDate,
		},
	}
}

// buildParentURI builds a parent URI for thread relationship.
func buildParentURI(msg *gmail.Message) *string {
	if msg.ThreadId != "" && msg.ThreadId != msg.Id {
		uri := fmt.Sprintf("gmail://threads/%s", msg.ThreadId)
		return &uri
	}
	return nil
}

// ShouldSyncMessage checks if a message should be synced based on config.
func ShouldSyncMessage(msg *gmail.Message, cfg *Config) bool {
	if !hasRequiredLabel(msg.LabelIds, cfg.LabelIDs) {
		return false
	}
	if !cfg.IncludeSpamTrash && isSpamOrTrash(msg.LabelIds) {
		return false
	}
	return true
}

// hasRequiredLabel checks if any required label is present.
func hasRequiredLabel(msgLabels, requiredLabels []string) bool {
	if len(requiredLabels) == 0 {
		return true
	}
	for _, required := range requiredLabels {
		for _, msgLabel := range msgLabels {
			if required == msgLabel {
				return true
			}
		}
	}
	return false
}

// isSpamOrTrash checks if the message has spam or trash labels.
func isSpamOrTrash(labels []string) bool {
	for _, label := range labels {
		if label == "SPAM" || label == "TRASH" {
			return true
		}
	}
	return false
}
