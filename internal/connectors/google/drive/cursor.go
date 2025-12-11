package drive

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("drive: invalid cursor format")

// Cursor tracks Google Drive sync state using the Changes API.
type Cursor struct {
	// Version is the cursor format version for future compatibility.
	Version int `json:"v"`
	// StartPageToken is the token from changes.getStartPageToken().
	// Used as the starting point for changes.list() in incremental sync.
	StartPageToken string `json:"start_page_token"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version: CursorVersion,
	}
}

// Encode serialises the cursor to a base64 string for storage.
func (c *Cursor) Encode() string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeCursor deserializes a cursor from a base64 string.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return NewCursor(), nil
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	// Version check for future migrations
	if cursor.Version > CursorVersion {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}

// IsEmpty returns true if the cursor has no sync state.
func (c *Cursor) IsEmpty() bool {
	return c.StartPageToken == ""
}
