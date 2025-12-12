package onedrive

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor stores the delta link for incremental sync.
type Cursor struct {
	Version   int    `json:"v"`
	DeltaLink string `json:"delta_link"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version: CursorVersion,
	}
}

// Encode serialises the cursor to a base64 string.
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

	if cursor.Version > CursorVersion {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}

// IsEmpty returns true if the cursor has no delta link.
func (c *Cursor) IsEmpty() bool {
	return c.DeltaLink == ""
}

// SetDeltaLink updates the delta link.
func (c *Cursor) SetDeltaLink(link string) {
	c.DeltaLink = link
}

// GetDeltaLink returns the current delta link.
func (c *Cursor) GetDeltaLink() string {
	return c.DeltaLink
}
