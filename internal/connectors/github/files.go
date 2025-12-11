package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	gh "github.com/google/go-github/v80/github"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// FetchFiles retrieves all files from a repository and converts them to RawDocuments.
func FetchFiles(
	ctx context.Context, client *Client, repo *gh.Repository, cfg *Config,
) ([]domain.RawDocument, string, error) {
	owner := repo.GetOwner().GetLogin()
	name := repo.GetName()
	branch := repo.GetDefaultBranch()

	// Get the tree
	tree, err := client.GetTree(ctx, owner, name, branch)
	if err != nil {
		return nil, "", err
	}

	// Filter to blobs only and apply patterns
	docs := make([]domain.RawDocument, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}

		path := entry.GetPath()

		// Check if file matches patterns
		if !matchesPatterns(path, cfg.FilePatterns) {
			continue
		}

		// Skip binary files by extension
		if isBinaryExtension(path) {
			continue
		}

		// Skip large files (> 1MB)
		if entry.GetSize() > 1024*1024 {
			continue
		}

		// Fetch blob content
		content, err := fetchBlobContent(ctx, client, owner, name, entry.GetSHA())
		if err != nil {
			// Skip files we can't read
			continue
		}

		// Create RawDocument
		doc := domain.RawDocument{
			SourceID: "", // Will be set by connector
			URI:      buildFileURI(owner, name, branch, path),
			MIMEType: detectFileMIMEType(path),
			Content:  content,
			Metadata: map[string]any{
				"type":   "file",
				"owner":  owner,
				"repo":   name,
				"branch": branch,
				"path":   path,
				"sha":    entry.GetSHA(),
				"size":   entry.GetSize(),
				"html_url": fmt.Sprintf(
					"https://github.com/%s/%s/blob/%s/%s",
					owner, name, branch, path,
				),
			},
		}
		docs = append(docs, doc)
	}

	return docs, tree.GetSHA(), nil
}

// fetchBlobContent fetches the content of a blob and decodes it.
func fetchBlobContent(ctx context.Context, client *Client, owner, repo, sha string) ([]byte, error) {
	blob, err := client.GetBlob(ctx, owner, repo, sha)
	if err != nil {
		return nil, err
	}

	// Decode base64 content
	if blob.GetEncoding() == "base64" {
		// Remove any whitespace from base64 content
		content := strings.ReplaceAll(blob.GetContent(), "\n", "")
		return base64.StdEncoding.DecodeString(content)
	}

	return []byte(blob.GetContent()), nil
}

// buildFileURI creates a URI for a file.
func buildFileURI(owner, repo, branch, path string) string {
	return fmt.Sprintf("github://%s/%s/blob/%s/%s", owner, repo, branch, path)
}

// extMIMETypes maps file extensions to MIME types for common types not in Go's registry.
var extMIMETypes = map[string]string{
	".md": "text/markdown", ".markdown": "text/markdown",
	".go": "text/x-go", ".py": "text/x-python", ".rs": "text/x-rust",
	".ts": "text/typescript", ".tsx": "text/typescript-jsx", ".jsx": "text/javascript-jsx",
	".yaml": "text/yaml", ".yml": "text/yaml", ".toml": "text/toml",
	".sh": "text/x-shellscript", ".bash": "text/x-shellscript",
	".sql": "text/x-sql", ".rb": "text/x-ruby", ".java": "text/x-java",
	".kt": "text/x-kotlin", ".kts": "text/x-kotlin",
	".swift": "text/x-swift", ".vue": "text/x-vue", ".svelte": "text/x-svelte",
}

// detectFileMIMEType determines the MIME type from file extension.
func detectFileMIMEType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "text/plain"
	}

	// Check our custom mappings first (avoids Go's mime returning video/mp2t for .ts)
	if t, ok := extMIMETypes[strings.ToLower(ext)]; ok {
		return t
	}

	// Fallback to Go's mime package
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		// Strip charset and other parameters.
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
		return mimeType
	}

	return "text/plain"
}

// matchesPatterns checks if a path matches any of the glob patterns.
func matchesPatterns(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		// Also try matching against full path
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// isBinaryExtension checks if a file extension indicates a binary file.
func isBinaryExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".7z": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".webp": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".pyc": true, ".pyo": true, ".class": true, ".o": true, ".a": true,
	}
	return binaryExts[ext]
}
