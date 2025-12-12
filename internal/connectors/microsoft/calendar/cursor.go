package calendar

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// CursorVersion is the current cursor format version.
const CursorVersion = 1

// ErrInvalidCursor indicates the cursor could not be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor stores delta links per calendar for incremental sync.
type Cursor struct {
	Version    int               `json:"v"`
	DeltaLinks map[string]string `json:"delta_links"`
}

// NewCursor creates a new empty cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Version:    CursorVersion,
		DeltaLinks: make(map[string]string),
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

	// Ensure DeltaLinks map is initialised
	if cursor.DeltaLinks == nil {
		cursor.DeltaLinks = make(map[string]string)
	}

	return &cursor, nil
}

// IsEmpty returns true if the cursor has no delta links.
func (c *Cursor) IsEmpty() bool {
	return len(c.DeltaLinks) == 0
}

// SetDeltaLink sets the delta link for a calendar.
func (c *Cursor) SetDeltaLink(calendarID, link string) {
	c.DeltaLinks[calendarID] = link
}

// GetDeltaLink returns the delta link for a calendar.
func (c *Cursor) GetDeltaLink(calendarID string) string {
	return c.DeltaLinks[calendarID]
}

// HasDeltaLink checks if a delta link exists for a calendar.
func (c *Cursor) HasDeltaLink(calendarID string) bool {
	_, ok := c.DeltaLinks[calendarID]
	return ok
}
