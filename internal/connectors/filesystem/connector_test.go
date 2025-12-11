package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNew(t *testing.T) {
	t.Run("creates connector with valid parameters", func(t *testing.T) {
		sourceID := "test-source-123"
		rootPath := "/tmp/test"

		connector := New(sourceID, rootPath)

		require.NotNil(t, connector)
		assert.Equal(t, sourceID, connector.sourceID)
		assert.Equal(t, rootPath, connector.rootPath)
	})

	t.Run("creates connector with empty strings", func(t *testing.T) {
		connector := New("", "")

		require.NotNil(t, connector)
		assert.Equal(t, "", connector.sourceID)
		assert.Equal(t, "", connector.rootPath)
	})

	t.Run("implements Connector interface", func(t *testing.T) {
		connector := New("test", "/tmp")
		var _ driven.Connector = connector
	})
}

func TestConnector_Type(t *testing.T) {
	t.Run("returns filesystem type", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		connType := connector.Type()

		assert.Equal(t, "filesystem", connType)
	})

	t.Run("type is consistent across multiple calls", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		type1 := connector.Type()
		type2 := connector.Type()

		assert.Equal(t, type1, type2)
	})
}

func TestConnector_SourceID(t *testing.T) {
	t.Run("returns correct source ID", func(t *testing.T) {
		expectedID := "my-source-id"
		connector := New(expectedID, "/tmp/test")

		sourceID := connector.SourceID()

		assert.Equal(t, expectedID, sourceID)
	})

	t.Run("returns empty string when source ID is empty", func(t *testing.T) {
		connector := New("", "/tmp/test")

		sourceID := connector.SourceID()

		assert.Equal(t, "", sourceID)
	})

	t.Run("source ID is consistent across multiple calls", func(t *testing.T) {
		connector := New("test-id", "/tmp/test")

		id1 := connector.SourceID()
		id2 := connector.SourceID()

		assert.Equal(t, id1, id2)
	})
}

func TestConnector_Capabilities(t *testing.T) {
	t.Run("returns expected capabilities", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		caps := connector.Capabilities()

		assert.True(t, caps.SupportsIncremental, "should support incremental sync")
		assert.True(t, caps.SupportsWatch, "should support watch")
		assert.True(t, caps.SupportsHierarchy, "should support hierarchy")
		assert.False(t, caps.SupportsBinary, "should not support binary")
	})

	t.Run("capabilities are consistent across multiple calls", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		caps1 := connector.Capabilities()
		caps2 := connector.Capabilities()

		assert.Equal(t, caps1, caps2)
	})
}

func TestConnector_FullSync(t *testing.T) {
	t.Run("syncs files from directory", func(t *testing.T) {
		// Create temp directory with files
		tempDir, err := os.MkdirTemp("", "sercha-test-fullsync-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create test files
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content 1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file2.md"), []byte("# Markdown"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		// Collect all documents
		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		// Check for errors
		select {
		case err, ok := <-errsChan:
			if ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		default:
		}

		assert.Len(t, docs, 2)
	})

	t.Run("skips hidden files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-hidden-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create visible and hidden files
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "visible.txt"), []byte("visible"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".hidden.txt"), []byte("hidden"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, _ := connector.FullSync(ctx)

		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		assert.Len(t, docs, 1)
		assert.Contains(t, docs[0].URI, "visible.txt")
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		connector := New("test-source", "/non/existent/path")
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		// Drain docs channel
		for range docsChan {
		}

		// Should receive an error
		select {
		case err := <-errsChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error for non-existent directory")
		}
	})

	t.Run("handles cancelled context", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-cancel-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		docsChan, errsChan := connector.FullSync(ctx)

		require.NotNil(t, docsChan)
		require.NotNil(t, errsChan)

		// Channels should close (sync may not start)
		for range docsChan {
		}
		for range errsChan {
		}
	})

	t.Run("includes file metadata", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-meta-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, _ := connector.FullSync(ctx)

		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		require.Len(t, docs, 1)
		doc := docs[0]

		assert.Equal(t, "test-source", doc.SourceID)
		assert.Contains(t, doc.URI, "test.txt")
		assert.Equal(t, "text/plain", doc.MIMEType)
		assert.Equal(t, []byte("hello"), doc.Content)
		assert.NotNil(t, doc.Metadata)
		assert.Equal(t, "test.txt", doc.Metadata["filename"])
		assert.Equal(t, "txt", doc.Metadata["extension"])
	})

	t.Run("detects MIME types correctly", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-mime-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		files := map[string]string{
			"file.md":   "text/markdown",
			"file.go":   "text/x-go",
			"file.py":   "text/x-python",
			"file.json": "application/json",
		}

		for name := range files {
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("content"), 0644))
		}

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, _ := connector.FullSync(ctx)

		docMap := make(map[string]string)
		for doc := range docsChan {
			docMap[filepath.Base(doc.URI)] = doc.MIMEType
		}

		for name, expectedMIME := range files {
			assert.Equal(t, expectedMIME, docMap[name], "MIME type mismatch for %s", name)
		}
	})
}

func TestConnector_IncrementalSync(t *testing.T) {
	t.Run("returns only modified files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-incr-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create files
		file1 := filepath.Join(tempDir, "old.txt")
		file2 := filepath.Join(tempDir, "new.txt")
		require.NoError(t, os.WriteFile(file1, []byte("old content"), 0644))

		// Wait a bit and record the time
		time.Sleep(50 * time.Millisecond)
		cursorTime := time.Now()

		// Create new file after cursor time
		time.Sleep(50 * time.Millisecond)
		require.NoError(t, os.WriteFile(file2, []byte("new content"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   fmt.Sprintf("%d", cursorTime.UnixNano()),
			LastSync: cursorTime,
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		var changes []domain.RawDocumentChange
		for change := range changesChan {
			changes = append(changes, change)
		}

		// Drain error channel
		for range errsChan {
		}

		// Should only get the new file
		assert.Len(t, changes, 1)
		if len(changes) > 0 {
			assert.Contains(t, changes[0].Document.URI, "new.txt")
		}
	})

	t.Run("handles empty cursor like full sync", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-incr-empty-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content 1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content 2"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   "", // Empty cursor
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		var changes []domain.RawDocumentChange
		for change := range changesChan {
			changes = append(changes, change)
		}

		for range errsChan {
		}

		// Should get all files
		assert.Len(t, changes, 2)
	})

	t.Run("handles invalid cursor format", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-incr-invalid-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx := context.Background()
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   "invalid-cursor-format",
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		for range changesChan {
		}

		select {
		case err := <-errsChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid cursor format")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error for invalid cursor")
		}
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		connector := New("test-source", "/non/existent/path")
		ctx := context.Background()
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   fmt.Sprintf("%d", time.Now().UnixNano()),
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		for range changesChan {
		}

		select {
		case err := <-errsChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not exist")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error for non-existent directory")
		}
	})

	t.Run("handles cancelled context", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-incr-cancel-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   fmt.Sprintf("%d", time.Now().UnixNano()),
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		require.NotNil(t, changesChan)
		require.NotNil(t, errsChan)

		// Channels should close
		for range changesChan {
		}
		for range errsChan {
		}
	})
}

func TestConnector_Watch(t *testing.T) {
	t.Run("watches for file changes", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-watch-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		changesChan, err := connector.Watch(ctx)
		require.NoError(t, err)
		require.NotNil(t, changesChan)

		// Create a file
		testFile := filepath.Join(tempDir, "new-file.txt")
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.WriteFile(testFile, []byte("content"), 0644)
		}()

		// Wait for event
		select {
		case change := <-changesChan:
			assert.Equal(t, domain.ChangeCreated, change.Type)
			assert.Contains(t, change.Document.URI, "new-file.txt")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timeout waiting for file change event")
		}

		// Clean up
		cancel()
		connector.Close()
	})

	t.Run("detects file modifications", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-watch-mod-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create initial file
		testFile := filepath.Join(tempDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("initial"), 0644))

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		changesChan, err := connector.Watch(ctx)
		require.NoError(t, err)

		// Modify the file
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.WriteFile(testFile, []byte("modified"), 0644)
		}()

		// Wait for event
		select {
		case change := <-changesChan:
			assert.Equal(t, domain.ChangeUpdated, change.Type)
			assert.Contains(t, change.Document.URI, "test.txt")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timeout waiting for file modification event")
		}

		cancel()
		connector.Close()
	})

	t.Run("detects file deletions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-watch-del-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create initial file
		testFile := filepath.Join(tempDir, "to-delete.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("delete me"), 0644))

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		changesChan, err := connector.Watch(ctx)
		require.NoError(t, err)

		// Delete the file
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.Remove(testFile)
		}()

		// Wait for event
		select {
		case change := <-changesChan:
			assert.Equal(t, domain.ChangeDeleted, change.Type)
			assert.Contains(t, change.Document.URI, "to-delete.txt")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timeout waiting for file deletion event")
		}

		cancel()
		connector.Close()
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		connector := New("test-source", "/non/existent/path")
		ctx := context.Background()

		changesChan, err := connector.Watch(ctx)

		assert.Error(t, err)
		assert.Nil(t, changesChan)
		assert.Contains(t, err.Error(), "root path error")
	})

	t.Run("closes channel when context is cancelled", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-watch-cancel-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())

		changesChan, err := connector.Watch(ctx)
		require.NoError(t, err)

		// Cancel context
		cancel()

		// Channel should close
		select {
		case _, ok := <-changesChan:
			if ok {
				// Got an event, wait for close
				for range changesChan {
				}
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("channel did not close after context cancellation")
		}

		connector.Close()
	})

	t.Run("returns error when connector is closed", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-watch-closed-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		connector.Close()

		ctx := context.Background()
		changesChan, err := connector.Watch(ctx)

		assert.Error(t, err)
		assert.Nil(t, changesChan)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestConnector_Close(t *testing.T) {
	t.Run("close succeeds", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		err := connector.Close()

		assert.NoError(t, err)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		err1 := connector.Close()
		err2 := connector.Close()
		err3 := connector.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
	})

	t.Run("basic operations after close still work", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		err := connector.Close()
		require.NoError(t, err)

		// Type and SourceID should still work
		assert.Equal(t, "filesystem", connector.Type())
		assert.Equal(t, "test-source", connector.SourceID())
		assert.NotNil(t, connector.Capabilities())
	})
}

func TestConnector_IntegrationWithTempDir(t *testing.T) {
	t.Run("connector works with real temp directory", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "sercha-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create some test files
		testFiles := []string{"file1.txt", "file2.md", "file3.json"}
		for _, fileName := range testFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Create connector with temp directory
		connector := New("test-source", tempDir)
		require.NotNil(t, connector)

		// Verify connector properties
		assert.Equal(t, "filesystem", connector.Type())
		assert.Equal(t, "test-source", connector.SourceID())
		assert.Equal(t, tempDir, connector.rootPath)

		// Verify capabilities
		caps := connector.Capabilities()
		assert.True(t, caps.SupportsIncremental)
		assert.True(t, caps.SupportsWatch)
		assert.True(t, caps.SupportsHierarchy)

		// Close connector
		err = connector.Close()
		assert.NoError(t, err)
	})

	t.Run("connector works with nested directories", func(t *testing.T) {
		// Create temporary directory with nested structure
		tempDir, err := os.MkdirTemp("", "sercha-test-nested-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create nested directories
		nestedPath := filepath.Join(tempDir, "dir1", "dir2", "dir3")
		err = os.MkdirAll(nestedPath, 0755)
		require.NoError(t, err)

		// Create files at different levels
		files := map[string]string{
			"root.txt":                  tempDir,
			"dir1/level1.txt":           filepath.Join(tempDir, "dir1"),
			"dir1/dir2/level2.txt":      filepath.Join(tempDir, "dir1", "dir2"),
			"dir1/dir2/dir3/level3.txt": filepath.Join(tempDir, "dir1", "dir2", "dir3"),
		}

		for fileName, dir := range files {
			filePath := filepath.Join(dir, filepath.Base(fileName))
			err := os.WriteFile(filePath, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Create connector with nested directory
		connector := New("nested-source", tempDir)
		require.NotNil(t, connector)

		// Verify it handles hierarchy
		caps := connector.Capabilities()
		assert.True(t, caps.SupportsHierarchy, "should support hierarchical structures")

		err = connector.Close()
		assert.NoError(t, err)
	})

	t.Run("connector handles non-existent directory", func(t *testing.T) {
		nonExistentPath := "/tmp/sercha-non-existent-dir-12345"

		// Ensure directory doesn't exist
		os.RemoveAll(nonExistentPath)

		// Connector should still be created (path validation happens later)
		connector := New("test-source", nonExistentPath)
		require.NotNil(t, connector)

		assert.Equal(t, nonExistentPath, connector.rootPath)
		assert.Equal(t, "test-source", connector.SourceID())

		err := connector.Close()
		assert.NoError(t, err)
	})

	t.Run("connector handles special characters in path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-special-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create directory with special characters (where allowed)
		specialDir := filepath.Join(tempDir, "dir-with_special.chars")
		err = os.MkdirAll(specialDir, 0755)
		require.NoError(t, err)

		connector := New("special-source", specialDir)
		require.NotNil(t, connector)

		assert.Equal(t, specialDir, connector.rootPath)

		err = connector.Close()
		assert.NoError(t, err)
	})
}

func TestConnector_ConcurrentOperations(t *testing.T) {
	t.Run("concurrent read operations are safe", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")
		ctx := context.Background()

		done := make(chan bool)

		// Run multiple operations concurrently
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				_ = connector.Type()
				_ = connector.SourceID()
				_ = connector.Capabilities()

				_, errs := connector.FullSync(ctx)
				<-errs // Drain error channel

				_, err := connector.Watch(ctx)
				assert.Error(t, err)
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		err := connector.Close()
		assert.NoError(t, err)
	})

	t.Run("concurrent close operations are safe", func(t *testing.T) {
		connector := New("test-source", "/tmp/test")

		done := make(chan bool)

		// Close concurrently
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				_ = connector.Close()
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestConnector_EdgeCases(t *testing.T) {
	t.Run("handles very long source ID", func(t *testing.T) {
		longID := string(make([]byte, 10000))
		for i := range longID {
			longID = longID[:i] + "a" + longID[i+1:]
		}

		connector := New(longID, "/tmp/test")
		require.NotNil(t, connector)

		assert.Equal(t, longID, connector.SourceID())
	})

	t.Run("handles very long path", func(t *testing.T) {
		longPath := "/tmp"
		for i := 0; i < 100; i++ {
			longPath = filepath.Join(longPath, "subdir")
		}

		connector := New("test-source", longPath)
		require.NotNil(t, connector)

		assert.Equal(t, longPath, connector.rootPath)
	})

	t.Run("handles unicode in source ID", func(t *testing.T) {
		unicodeID := "test-源-🚀-مرحبا"

		connector := New(unicodeID, "/tmp/test")
		require.NotNil(t, connector)

		assert.Equal(t, unicodeID, connector.SourceID())
	})

	t.Run("handles unicode in path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create directory with unicode name (where supported by OS)
		unicodeDir := filepath.Join(tempDir, "测试目录")
		err = os.MkdirAll(unicodeDir, 0755)
		if err == nil {
			connector := New("test-source", unicodeDir)
			require.NotNil(t, connector)
			assert.Equal(t, unicodeDir, connector.rootPath)
		}
		// Skip if OS doesn't support unicode paths
	})
}

// TestConnector_Validate tests the Validate function with various scenarios.
func TestConnector_Validate(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) (string, func())
		expectError   bool
		errorContains string
	}{
		{
			name: "valid directory succeeds",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "sercha-validate-*")
				require.NoError(t, err)
				return tempDir, func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
		},
		{
			name: "non-existent path returns error",
			setup: func(t *testing.T) (string, func()) {
				return "/non/existent/path/12345", func() {}
			},
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name: "file instead of directory returns error",
			setup: func(t *testing.T) (string, func()) {
				tempDir, err := os.MkdirTemp("", "sercha-validate-file-*")
				require.NoError(t, err)
				filePath := filepath.Join(tempDir, "file.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))
				return filePath, func() { os.RemoveAll(tempDir) }
			},
			expectError:   true,
			errorContains: "not a directory",
		},
		// NOTE: Permission denied test removed as it is unreliable across platforms.
		// macOS and some Linux configurations do not enforce 0000 permissions on
		// directories when running as a normal user or with certain security settings.
		// The permission handling code is tested implicitly via integration tests.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setup(t)
			defer cleanup()

			connector := New("test-source", path)
			ctx := context.Background()

			err := connector.Validate(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("context cancellation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-validate-ctx-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = connector.Validate(ctx)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// TestDetectMIMEType tests the detectMIMEType function with various file extensions.
func TestDetectMIMEType(t *testing.T) {
	tests := []struct {
		filename     string
		expectedMIME string
	}{
		// No extension
		{"file", "text/plain"},
		{"noext", "text/plain"},

		// Custom fallback types
		{"doc.md", "text/markdown"},
		{"doc.markdown", "text/markdown"},
		{"code.go", "text/x-go"},
		{"script.py", "text/x-python"},
		{"lib.rs", "text/x-rust"},
		{"app.ts", "text/typescript"},
		{"component.tsx", "text/typescript-jsx"},
		{"component.jsx", "text/javascript-jsx"},
		{"config.yaml", "text/yaml"},
		{"config.yml", "text/yaml"},
		{"config.toml", "text/toml"},
		{"script.sh", "text/x-shellscript"},
		{"script.bash", "text/x-shellscript"},
		{"query.sql", "text/x-sql"},

		// Standard MIME types (from Go's mime package)
		{"data.json", "application/json"},
		{"page.html", "text/html"},
		{"style.css", "text/css"},
		{"script.js", "text/javascript"},
		{"data.xml", "application/xml"}, // macOS returns application/xml
		{"doc.pdf", "application/pdf"},
		{"archive.zip", "application/zip"},
		{"image.png", "image/png"},
		{"image.jpg", "image/jpeg"},
		{"image.gif", "image/gif"},

		// Unknown extension - use truly obscure extensions to avoid platform MIME registrations
		{"file.zzzzunknown", "application/octet-stream"},
		{"file.xyzabc123", "application/octet-stream"},

		// Case insensitive
		{"FILE.MD", "text/markdown"},
		{"FILE.GO", "text/x-go"},
		{"FILE.PY", "text/x-python"},

		// Mixed case
		{"File.Yaml", "text/yaml"},
		{"File.Toml", "text/toml"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			mimeType := detectMIMEType(tt.filename)
			assert.Equal(t, tt.expectedMIME, mimeType)
		})
	}

	t.Run("strips charset from mime type", func(t *testing.T) {
		// Test with files that might have charset
		testFiles := []string{"file.html", "file.css", "file.js"}
		for _, file := range testFiles {
			mimeType := detectMIMEType(file)
			// Should not contain charset parameter
			assert.NotContains(t, mimeType, "charset")
			assert.NotContains(t, mimeType, ";")
		}
	})
}

// TestIsHidden tests the isHidden function with various path scenarios.
func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Hidden files
		{".hidden", ".hidden", true},
		{".hiddenfile", ".hiddenfile", true},
		{"path/to/.hidden", "path/to/.hidden", true},
		{"/root/.config", "/root/.config", true},
		{"/root/.config/file.txt", "/root/.config/file.txt", true},

		// Hidden directories in path
		{"/path/.hidden/file.txt", "/path/.hidden/file.txt", true},
		{"dir/.git/config", "dir/.git/config", true},
		{"/home/user/.ssh/id_rsa", "/home/user/.ssh/id_rsa", true},

		// Not hidden
		{"file.txt", "file.txt", false},
		{"path/to/file.txt", "path/to/file.txt", false},
		{"/root/visible/file.txt", "/root/visible/file.txt", false},
		{"normal.file", "normal.file", false},

		// Special cases - . and .. are not considered hidden
		{".", ".", false},
		{"..", "..", false},
		{"path/./file", "path/./file", false},
		{"path/../file", "path/../file", false},

		// Edge cases
		{"", "", false},
		{"/", "/", false},
		{"file.hidden", "file.hidden", false}, // Dot in filename but not prefix
		{"directory.name/file", "directory.name/file", false},

		// Multiple hidden directories
		{".config/.cache/data", ".config/.cache/data", true},
		{"/a/.b/.c/file", "/a/.b/.c/file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHidden(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHandleFsEvent tests the handleFsEvent function with various event types.
func TestHandleFsEvent(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      bool
		setupDir       bool
		setupHidden    bool
		operation      fsnotify.Op
		expectedChange bool
		expectedType   domain.ChangeType
	}{
		{
			name:           "create file event",
			setupFile:      true,
			operation:      fsnotify.Create,
			expectedChange: true,
			expectedType:   domain.ChangeCreated,
		},
		{
			name:           "write file event",
			setupFile:      true,
			operation:      fsnotify.Write,
			expectedChange: true,
			expectedType:   domain.ChangeUpdated,
		},
		{
			name:           "remove file event",
			setupFile:      false, // File doesn't exist (already removed)
			operation:      fsnotify.Remove,
			expectedChange: true,
			expectedType:   domain.ChangeDeleted,
		},
		{
			name:           "rename file event",
			setupFile:      false, // Old file doesn't exist
			operation:      fsnotify.Rename,
			expectedChange: true,
			expectedType:   domain.ChangeDeleted,
		},
		{
			name:           "chmod file event - not handled",
			setupFile:      true,
			operation:      fsnotify.Chmod,
			expectedChange: false,
		},
		{
			name:           "create directory event - should be skipped",
			setupDir:       true,
			operation:      fsnotify.Create,
			expectedChange: false,
		},
		{
			name:           "hidden file create - should be skipped",
			setupHidden:    true,
			operation:      fsnotify.Create,
			expectedChange: false,
		},
		{
			name:           "hidden file write - should be skipped",
			setupHidden:    true,
			operation:      fsnotify.Write,
			expectedChange: false,
		},
		{
			name:           "hidden file remove - should be skipped",
			setupHidden:    true,
			operation:      fsnotify.Remove,
			expectedChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "sercha-event-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			var eventPath string

			if tt.setupDir {
				eventPath = filepath.Join(tempDir, "testdir")
				require.NoError(t, os.Mkdir(eventPath, 0755))
			} else if tt.setupHidden {
				eventPath = filepath.Join(tempDir, ".hidden.txt")
				if tt.operation != fsnotify.Remove {
					require.NoError(t, os.WriteFile(eventPath, []byte("hidden"), 0644))
				}
			} else if tt.setupFile {
				eventPath = filepath.Join(tempDir, "test.txt")
				require.NoError(t, os.WriteFile(eventPath, []byte("content"), 0644))
			} else {
				eventPath = filepath.Join(tempDir, "removed.txt")
			}

			connector := New("test-source", tempDir)
			event := fsnotify.Event{
				Name: eventPath,
				Op:   tt.operation,
			}

			change := connector.handleFsEvent(event)

			if tt.expectedChange {
				require.NotNil(t, change, "expected change but got nil")
				assert.Equal(t, tt.expectedType, change.Type)
				assert.Equal(t, eventPath, change.Document.URI)
				assert.Equal(t, "test-source", change.Document.SourceID)

				// For non-delete operations, check content was read
				if tt.expectedType != domain.ChangeDeleted && tt.setupFile {
					assert.NotEmpty(t, change.Document.Content)
				}
			} else {
				assert.Nil(t, change, "expected no change but got one")
			}
		})
	}

	t.Run("combined operations", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-event-combined-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		testFile := filepath.Join(tempDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

		connector := New("test-source", tempDir)

		// Test combined operations (e.g., Write | Chmod)
		event := fsnotify.Event{
			Name: testFile,
			Op:   fsnotify.Write | fsnotify.Chmod,
		}

		change := connector.handleFsEvent(event)

		// Should handle Write operation
		require.NotNil(t, change)
		assert.Equal(t, domain.ChangeUpdated, change.Type)
	})
}

// TestConnector_FullSync_EdgeCases tests additional edge cases for FullSync.
func TestConnector_FullSync_EdgeCases(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-fullsync-empty-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		for range errsChan {
		}

		// Empty directory should yield no documents
		assert.Empty(t, docs)
	})

	t.Run("directory with only hidden files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-fullsync-hidden-only-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create only hidden files
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".hidden1"), []byte("h1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".hidden2"), []byte("h2"), 0644))

		// Create hidden directory
		hiddenDir := filepath.Join(tempDir, ".hiddendir")
		require.NoError(t, os.Mkdir(hiddenDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "file.txt"), []byte("hidden"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		for range errsChan {
		}

		// Should not include any hidden files
		assert.Empty(t, docs)
	})

	t.Run("context cancellation during walk", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-fullsync-cancel-walk-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create many files to increase chance of cancellation during walk
		for i := 0; i < 100; i++ {
			require.NoError(t, os.WriteFile(
				filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i)),
				[]byte(fmt.Sprintf("content %d", i)),
				0644,
			))
		}

		connector := New("test-source", tempDir)
		ctx, cancel := context.WithCancel(context.Background())

		docsChan, errsChan := connector.FullSync(ctx)

		// Cancel after receiving first few documents
		docCount := 0
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		for range docsChan {
			docCount++
		}

		for range errsChan {
		}

		// Should have been cancelled before processing all files
		t.Logf("Processed %d documents before cancellation", docCount)
	})

	t.Run("file path is a file not directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-fullsync-file-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "notadir.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

		connector := New("test-source", filePath)
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		// Drain docs
		for range docsChan {
		}

		// Should get error
		select {
		case err := <-errsChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not a directory")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error for file path")
		}
	})

	t.Run("handles subdirectories correctly", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-fullsync-subdir-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create nested structure
		subdir1 := filepath.Join(tempDir, "subdir1")
		subdir2 := filepath.Join(tempDir, "subdir2")
		nested := filepath.Join(subdir1, "nested")
		require.NoError(t, os.MkdirAll(nested, 0755))
		require.NoError(t, os.Mkdir(subdir2, 0755))

		// Create files at different levels
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("r"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(subdir1, "file1.txt"), []byte("f1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(subdir2, "file2.txt"), []byte("f2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(nested, "file3.txt"), []byte("f3"), 0644))

		connector := New("test-source", tempDir)
		ctx := context.Background()

		docsChan, errsChan := connector.FullSync(ctx)

		var docs []domain.RawDocument
		for doc := range docsChan {
			docs = append(docs, doc)
		}

		for range errsChan {
		}

		// Should find all 4 files
		assert.Len(t, docs, 4)

		// Verify parent URI is set correctly
		for _, doc := range docs {
			if filepath.Base(doc.URI) == "root.txt" {
				// Root level file should have no parent
				assert.Nil(t, doc.ParentURI)
			} else {
				// Nested files should have parent
				assert.NotNil(t, doc.ParentURI)
			}
		}
	})
}

// TestConnector_IncrementalSync_EdgeCases tests additional edge cases for IncrementalSync.
func TestConnector_IncrementalSync_EdgeCases(t *testing.T) {
	t.Run("cursor with exact file modification time", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-incr-exact-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create file and get its exact modification time
		testFile := filepath.Join(tempDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		exactTime := info.ModTime()

		connector := New("test-source", tempDir)
		ctx := context.Background()

		// Use exact modification time as cursor
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   fmt.Sprintf("%d", exactTime.UnixNano()),
			LastSync: exactTime,
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		var changes []domain.RawDocumentChange
		for change := range changesChan {
			changes = append(changes, change)
		}

		for range errsChan {
		}

		// File with exact same time IS included (not before sinceTime).
		// This prevents missing files modified at exactly the sync boundary.
		assert.Len(t, changes, 1)
		assert.Equal(t, testFile, changes[0].Document.URI)
	})

	t.Run("returns SyncComplete with new cursor", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-incr-cursor-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx := context.Background()

		beforeSync := time.Now()

		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   "",
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		// Drain changes
		for range changesChan {
		}

		// Check for SyncComplete error
		var gotSyncComplete bool
		for err := range errsChan {
			if syncComplete, ok := err.(*driven.SyncComplete); ok {
				gotSyncComplete = true
				assert.NotEmpty(t, syncComplete.NewCursor)

				// Parse cursor and verify it's recent
				cursorNanos, parseErr := strconv.ParseInt(syncComplete.NewCursor, 10, 64)
				require.NoError(t, parseErr)
				cursorTime := time.Unix(0, cursorNanos)

				assert.True(t, cursorTime.After(beforeSync) || cursorTime.Equal(beforeSync))
			}
		}

		assert.True(t, gotSyncComplete, "should receive SyncComplete")
	})

	t.Run("directory path is a file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-incr-file-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "notadir.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

		connector := New("test-source", filePath)
		ctx := context.Background()

		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   "",
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		for range changesChan {
		}

		select {
		case err := <-errsChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not a directory")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error")
		}
	})

	t.Run("negative cursor value", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sercha-incr-neg-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		connector := New("test-source", tempDir)
		ctx := context.Background()

		// Negative timestamp (still valid int64)
		syncState := domain.SyncState{
			SourceID: "test-source",
			Cursor:   "-1000",
		}

		changesChan, errsChan := connector.IncrementalSync(ctx, syncState)

		// Should work with negative timestamp (represents time before epoch)
		for range changesChan {
		}

		for range errsChan {
		}
	})
}
