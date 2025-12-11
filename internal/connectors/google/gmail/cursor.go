package gmail

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("gmail: invalid cursor format")

// Cursor tracks Gmail sync state using the History API.
type Cursor struct {
	// Version is the cursor format version for future compatibility.
	Version int `json:"v"`
	// HistoryID is the history ID from the last sync.
	// Used as the starting point for history.list() in incremental sync.
	HistoryID uint64 `json:"history_id"`
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
	return c.HistoryID == 0
}
