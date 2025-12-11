package gmail

import "strings"

// ResolveWebURL converts a Gmail URI to a web URL.
// gmail://messages/{id} -> https://mail.google.com/mail/u/0/#all/{id}
func ResolveWebURL(uri string, _ map[string]any) string {
	if strings.HasPrefix(uri, "gmail://messages/") {
		messageID := strings.TrimPrefix(uri, "gmail://messages/")
		return "https://mail.google.com/mail/u/0/#all/" + messageID
	}
	return ""
}
