package normalisers

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/docx"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/eml"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/github"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/html"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/ics"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/markdown"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/notion"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/pdf"
	"github.com/custodia-labs/sercha-cli/internal/normalisers/plaintext"
)

// Ensure Registry implements the interface.
var _ driven.NormaliserRegistry = (*Registry)(nil)

// Registry manages normaliser registrations.
type Registry struct {
	mu          sync.RWMutex
	normalisers []driven.Normaliser
	byMIME      map[string][]driven.Normaliser
}

// NewRegistry creates a new normaliser registry with default normalisers.
func NewRegistry() *Registry {
	r := &Registry{
		normalisers: make([]driven.Normaliser, 0),
		byMIME:      make(map[string][]driven.Normaliser),
	}
	// Register default normalisers
	r.Register(docx.New())
	r.Register(eml.New())
	r.Register(html.New())
	r.Register(ics.New())
	r.Register(markdown.New())
	r.Register(pdf.New())
	r.Register(plaintext.New())

	// Register GitHub-specific normalisers
	r.Register(github.NewIssue())
	r.Register(github.NewPull())

	// Register Notion-specific normalisers
	r.Register(notion.NewPage())
	r.Register(notion.NewDatabase())
	r.Register(notion.NewDatabaseItem())

	return r
}

// Normalise transforms a raw document using the best matching normaliser.
func (r *Registry) Normalise(ctx context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	r.mu.RLock()
	candidates := r.byMIME[raw.MIMEType]
	r.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no normaliser for MIME type %q: %w", raw.MIMEType, domain.ErrNotImplemented)
	}

	// Candidates are already sorted by priority
	return candidates[0].Normalise(ctx, raw)
}

// Register adds a normaliser to the registry.
func (r *Registry) Register(n driven.Normaliser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.normalisers = append(r.normalisers, n)
	for _, mime := range n.SupportedMIMETypes() {
		r.byMIME[mime] = append(r.byMIME[mime], n)
		// Sort by priority descending
		sort.Slice(r.byMIME[mime], func(i, j int) bool {
			return r.byMIME[mime][i].Priority() > r.byMIME[mime][j].Priority()
		})
	}
}

// SupportedMIMETypes returns all MIME types that can be normalised.
func (r *Registry) SupportedMIMETypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.byMIME))
	for mime := range r.byMIME {
		types = append(types, mime)
	}
	return types
}
