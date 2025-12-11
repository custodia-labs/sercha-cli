// Package mcp provides an MCP (Model Context Protocol) server adapter for Sercha.
// It enables AI assistants like Claude to interact with Sercha's local search capabilities.
package mcp

import "errors"

// ErrMissingSearchService is returned when the search service is not provided.
var ErrMissingSearchService = errors.New("mcp: search service is required")
