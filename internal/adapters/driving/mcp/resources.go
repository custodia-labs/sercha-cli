package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// URIScheme is the custom URI scheme for Sercha resources.
	uriScheme = "sercha://"
)

// registerResources registers all resource handlers with the MCP server.
func (s *Server) registerResources() {
	// Static resource for listing sources.
	s.server.AddResource(&mcp.Resource{
		URI:         uriScheme + "sources",
		Name:        "sources",
		Description: "List of all configured data sources",
		MIMEType:    "application/json",
	}, s.handleSourcesResource)

	// Template for source documents.
	s.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "sources/{sourceId}/documents",
		Name:        "source-documents",
		Description: "Documents indexed from a specific source",
		MIMEType:    "application/json",
	}, s.handleDocumentsResource)

	// Template for document content.
	s.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: uriScheme + "documents/{documentId}",
		Name:        "document-content",
		Description: "Content of a specific document",
		MIMEType:    "text/plain",
	}, s.handleDocumentContentResource)
}

// handleSourcesResource returns a list of all configured sources.
func (s *Server) handleSourcesResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if s.ports.Source == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     "[]",
			}},
		}, nil
	}

	sources, err := s.ports.Source.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	// Build simplified source list.
	type sourceInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
		URI  string `json:"uri"`
	}

	infos := make([]sourceInfo, len(sources))
	for i, src := range sources {
		// Get path from config if available (filesystem sources).
		uri := ""
		if path, ok := src.Config["path"]; ok {
			uri = path
		}
		infos[i] = sourceInfo{
			ID:   src.ID,
			Name: src.Name,
			Type: src.Type,
			URI:  uri,
		}
	}

	data, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling sources: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// handleDocumentsResource returns documents for a specific source.
func (s *Server) handleDocumentsResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if s.ports.Document == nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Extract sourceId from URI: sercha://sources/{sourceId}/documents
	sourceID := extractSourceID(req.Params.URI)
	if sourceID == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	docs, err := s.ports.Document.ListBySource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("listing documents: %w", err)
	}

	// Build simplified document list.
	type docInfo struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		URI   string `json:"uri"`
	}

	infos := make([]docInfo, len(docs))
	for i := range docs {
		infos[i] = docInfo{
			ID:    docs[i].ID,
			Title: docs[i].Title,
			URI:   docs[i].URI,
		}
	}

	data, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling documents: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// handleDocumentContentResource returns the content of a specific document.
func (s *Server) handleDocumentContentResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if s.ports.Document == nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Extract documentId from URI: sercha://documents/{documentId}
	docID := extractDocumentID(req.Params.URI)
	if docID == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	content, err := s.ports.Document.GetContent(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("getting document content: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     content,
		}},
	}, nil
}

// extractSourceID extracts the source ID from a URI like sercha://sources/{sourceId}/documents.
func extractSourceID(uri string) string {
	const prefix = uriScheme + "sources/"
	const suffix = "/documents"

	if !strings.HasPrefix(uri, prefix) {
		return ""
	}

	uri = strings.TrimPrefix(uri, prefix)
	if !strings.HasSuffix(uri, suffix) {
		return ""
	}

	return strings.TrimSuffix(uri, suffix)
}

// extractDocumentID extracts the document ID from a URI like sercha://documents/{documentId}.
func extractDocumentID(uri string) string {
	const prefix = uriScheme + "documents/"

	if !strings.HasPrefix(uri, prefix) {
		return ""
	}

	return strings.TrimPrefix(uri, prefix)
}
