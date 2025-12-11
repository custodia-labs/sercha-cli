package domain

import "strings"

// AuthCapability represents supported authentication capabilities for a provider.
// This is a bitfield allowing providers to support multiple auth methods.
type AuthCapability uint8

const (
	// AuthCapNone indicates no authentication is needed.
	AuthCapNone AuthCapability = 0
	// AuthCapPAT indicates Personal Access Token authentication is supported.
	AuthCapPAT AuthCapability = 1 << 0
	// AuthCapOAuth indicates OAuth 2.0 authentication is supported.
	AuthCapOAuth AuthCapability = 1 << 1
)

// SupportsPAT returns true if PAT authentication is supported.
func (c AuthCapability) SupportsPAT() bool {
	return c&AuthCapPAT != 0
}

// SupportsOAuth returns true if OAuth authentication is supported.
func (c AuthCapability) SupportsOAuth() bool {
	return c&AuthCapOAuth != 0
}

// SupportsMultipleMethods returns true if more than one auth method is supported.
func (c AuthCapability) SupportsMultipleMethods() bool {
	return c.SupportsPAT() && c.SupportsOAuth()
}

// RequiresAuth returns true if any authentication is required.
func (c AuthCapability) RequiresAuth() bool {
	return c != AuthCapNone
}

// SupportedMethods returns a slice of supported AuthMethods.
// Returns an empty slice if no authentication is required.
func (c AuthCapability) SupportedMethods() []AuthMethod {
	var methods []AuthMethod
	if c.SupportsPAT() {
		methods = append(methods, AuthMethodPAT)
	}
	if c.SupportsOAuth() {
		methods = append(methods, AuthMethodOAuth)
	}
	return methods
}

// String returns a human-readable representation.
func (c AuthCapability) String() string {
	if c == AuthCapNone {
		return "none"
	}
	var parts []string
	if c.SupportsPAT() {
		parts = append(parts, "pat")
	}
	if c.SupportsOAuth() {
		parts = append(parts, "oauth")
	}
	return strings.Join(parts, ",")
}
