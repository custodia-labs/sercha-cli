package outlook

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("outlook: invalid cursor format")

// Cursor tracks Outlook sync state using Microsoft Graph delta queries.
type Cursor struct {
	// Version is the cursor format version for future compatibility.
	Version int `json:"v"`
	// DeltaLink is the delta link URL from Microsoft Graph.
	// Used to fetch only changes since the last sync.
	DeltaLink string `json:"delta_link"`
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

// DecodeCursor deserialises a cursor from a base64 string.
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
	return c.DeltaLink == ""
}

// SetDeltaLink updates the delta link.
func (c *Cursor) SetDeltaLink(deltaLink string) {
	c.DeltaLink = deltaLink
}

// GetDeltaLink returns the delta link.
func (c *Cursor) GetDeltaLink() string {
	return c.DeltaLink
}
