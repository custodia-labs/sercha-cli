package services

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Operating system identifiers.
const (
	osDarwin  = "darwin"
	osLinux   = "linux"
	osWindows = "windows"
)

// Ensure ResultActionService implements the interface.
var _ driving.ResultActionService = (*ResultActionService)(nil)

// ResultActionService provides actions on search results.
type ResultActionService struct {
	sourceStore       driven.SourceStore
	connectorRegistry driving.ConnectorRegistry
}

// NewResultActionService creates a new result action service.
func NewResultActionService(
	sourceStore driven.SourceStore,
	connectorRegistry driving.ConnectorRegistry,
) *ResultActionService {
	return &ResultActionService{
		sourceStore:       sourceStore,
		connectorRegistry: connectorRegistry,
	}
}

// CopyToClipboard copies the result's content to the system clipboard.
func (s *ResultActionService) CopyToClipboard(_ context.Context, result *domain.SearchResult) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}

	// Copy the chunk content to clipboard
	content := result.Chunk.Content
	return copyToClipboard(content)
}

// OpenDocument opens the result's document in the default application.
func (s *ResultActionService) OpenDocument(ctx context.Context, result *domain.SearchResult) error {
	if result == nil {
		return fmt.Errorf("result is nil")
	}

	// Resolve the URL using the connector's resolver
	openableURL := s.resolveWebURL(ctx, &result.Document)

	// Open the resolved URL
	return openURL(openableURL)
}

// resolveWebURL converts a document URI to an openable URL using the connector's resolver.
func (s *ResultActionService) resolveWebURL(ctx context.Context, doc *domain.Document) string {
	if resolved := s.tryConnectorResolver(ctx, doc); resolved != "" {
		return resolved
	}
	return convertToOpenableURL(doc.URI)
}

// tryConnectorResolver attempts to resolve URL using the connector's WebURLResolver.
func (s *ResultActionService) tryConnectorResolver(ctx context.Context, doc *domain.Document) string {
	if s.sourceStore == nil || s.connectorRegistry == nil {
		return ""
	}
	source, err := s.sourceStore.Get(ctx, doc.SourceID)
	if err != nil || source == nil {
		return ""
	}
	connectorType, err := s.connectorRegistry.Get(source.Type)
	if err != nil || connectorType == nil || connectorType.WebURLResolver == nil {
		return ""
	}
	return connectorType.WebURLResolver(doc.URI, doc.Metadata)
}

// copyToClipboard copies text to the system clipboard using OS-specific commands.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case osDarwin:
		cmd = exec.Command("pbcopy")
	case osLinux:
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	case osWindows:
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
