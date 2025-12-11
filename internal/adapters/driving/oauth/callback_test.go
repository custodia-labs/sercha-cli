//nolint:noctx // Test file uses http.Get for convenience; context not required in tests
package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for CallbackServer

func TestNewCallbackServer(t *testing.T) {
	port := 8080
	state := "test-state-123"

	server := NewCallbackServer(port, state)

	require.NotNil(t, server)
	assert.Equal(t, port, server.port)
	assert.Equal(t, state, server.expectedState)
	assert.NotNil(t, server.codeChan)
	assert.NotNil(t, server.errChan)
	assert.Nil(t, server.server)
	assert.Nil(t, server.listener)
}

func TestCallbackServer_Start(t *testing.T) {
	// Find an available port to avoid conflicts
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)

	// Verify server is running
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.listener)

	// Clean up
	err = server.Stop()
	require.NoError(t, err)
}

func TestCallbackServer_Start_PortInUse(t *testing.T) {
	// Find an available port
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	// Start first server
	server1 := NewCallbackServer(port, "test-state-1")
	err = server1.Start()
	require.NoError(t, err)
	defer server1.Stop()

	// Try to start second server on same port
	server2 := NewCallbackServer(port, "test-state-2")
	err = server2.Start()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

func TestCallbackServer_Stop(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)

	err = server.Stop()
	require.NoError(t, err)

	// Stopping again should not error
	err = server.Stop()
	require.NoError(t, err)
}

func TestCallbackServer_Stop_NotStarted(t *testing.T) {
	server := NewCallbackServer(8080, "test-state")

	// Should not error when stopping a server that was never started
	err := server.Stop()
	require.NoError(t, err)
}

func TestCallbackServer_Port(t *testing.T) {
	expectedPort := 9090
	server := NewCallbackServer(expectedPort, "test-state")

	actualPort := server.Port()

	assert.Equal(t, expectedPort, actualPort)
}

func TestCallbackServer_RedirectURI(t *testing.T) {
	port := 9090
	server := NewCallbackServer(port, "test-state")

	redirectURI := server.RedirectURI()

	expected := fmt.Sprintf("http://localhost:%d/callback", port)
	assert.Equal(t, expected, redirectURI)
}

func TestCallbackServer_HandleCallback_Success(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "test-state-abc123"
	expectedCode := "auth-code-xyz789"

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=%s&state=%s",
		port, expectedCode, expectedState))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	// Wait for code to be received
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case code := <-server.codeChan:
		assert.Equal(t, expectedCode, code)
	case <-ctx.Done():
		t.Fatal("timeout waiting for code")
	}
}

func TestCallbackServer_HandleCallback_StateMismatch(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "correct-state"
	wrongState := "wrong-state"

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request with wrong state
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=somecode&state=%s",
		port, wrongState))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK but with error in channel
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for error to be received
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case err := <-server.errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state mismatch")
		assert.Contains(t, err.Error(), expectedState)
		assert.Contains(t, err.Error(), wrongState)
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestCallbackServer_HandleCallback_MissingCode(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "test-state"

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request without code
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?state=%s",
		port, expectedState))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK but with error in channel
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for error to be received
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case err := <-server.errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no authorization code received")
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestCallbackServer_HandleCallback_OAuthError(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request with OAuth error
	errorCode := "access_denied"
	errorDesc := "User denied access"

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?error=%s&error_description=%s",
		port, url.QueryEscape(errorCode), url.QueryEscape(errorDesc)))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK but with error in channel
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for error to be received
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case err := <-server.errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "oauth error")
		assert.Contains(t, err.Error(), errorCode)
		assert.Contains(t, err.Error(), errorDesc)
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestCallbackServer_HandleCallback_EmptyState(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "expected-state"

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request with empty state
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=somecode&state=", port))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK but with error in channel
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for error to be received
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case err := <-server.errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state mismatch")
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestCallbackServer_WaitForCode_Success(t *testing.T) {
	server := NewCallbackServer(8080, "test-state")
	expectedCode := "auth-code-123"

	// Send code in goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		server.codeChan <- expectedCode
	}()

	code, err := server.WaitForCode(5 * time.Second)

	require.NoError(t, err)
	assert.Equal(t, expectedCode, code)
}

func TestCallbackServer_WaitForCode_Error(t *testing.T) {
	server := NewCallbackServer(8080, "test-state")
	expectedError := fmt.Errorf("oauth error occurred")

	// Send error in goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		server.errChan <- expectedError
	}()

	code, err := server.WaitForCode(5 * time.Second)

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, code)
}

func TestCallbackServer_WaitForCode_Timeout(t *testing.T) {
	server := NewCallbackServer(8080, "test-state")

	// Don't send anything - should timeout
	code, err := server.WaitForCode(100 * time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for authorization callback")
	assert.Empty(t, code)
}

func TestCallbackServer_WaitForCode_MultipleWaiters(t *testing.T) {
	server := NewCallbackServer(8080, "test-state")
	expectedCode := "auth-code-multi"

	// Send code once
	go func() {
		time.Sleep(50 * time.Millisecond)
		server.codeChan <- expectedCode
	}()

	// First waiter should get the code
	code1, err1 := server.WaitForCode(5 * time.Second)
	require.NoError(t, err1)
	assert.Equal(t, expectedCode, code1)

	// Second waiter should timeout (code already consumed)
	code2, err2 := server.WaitForCode(100 * time.Millisecond)
	require.Error(t, err2)
	assert.Contains(t, err2.Error(), "timeout")
	assert.Empty(t, code2)
}

func TestCallbackServer_ConcurrentCallbacks(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "concurrent-state"
	server := NewCallbackServer(port, expectedState)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make multiple concurrent callback requests
	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			code := fmt.Sprintf("code-%d", index)
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=%s&state=%s",
				port, code, expectedState))
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()

	// At least one code should have been received
	// (Only the first one gets through due to buffered channel of size 1)
	select {
	case code := <-server.codeChan:
		assert.NotEmpty(t, code)
	case <-time.After(1 * time.Second):
		t.Fatal("no code received")
	}
}

func TestCallbackServer_ThreadSafety(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	// Start and stop concurrently
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = server.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		_ = server.Stop()
	}()

	wg.Wait()

	// Should be able to start again
	err = server.Start()
	require.NoError(t, err)

	err = server.Stop()
	require.NoError(t, err)
}

// Tests for successHTML

func TestSuccessHTML(t *testing.T) {
	title := "Test Title"
	message := "Test Message"

	html := successHTML(title, message)

	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, title)
	assert.Contains(t, html, message)
	assert.Contains(t, html, "Sercha - OAuth Callback")
	// Verify HTML structure
	assert.Contains(t, html, "<html>")
	assert.Contains(t, html, "<head>")
	assert.Contains(t, html, "<body>")
	assert.Contains(t, html, "<style>")
}

func TestSuccessHTML_SpecialCharacters(t *testing.T) {
	title := "Success & Complete"
	message := "You are all set!"

	html := successHTML(title, message)

	// Should still generate valid HTML
	assert.Contains(t, html, title)
	assert.Contains(t, html, message)
	assert.Contains(t, html, "<!DOCTYPE html>")
}

func TestSuccessHTML_EmptyStrings(t *testing.T) {
	html := successHTML("", "")

	// Should still generate valid HTML structure
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<html>")
	assert.Contains(t, html, "<h1></h1>")
	assert.Contains(t, html, "<p></p>")
}

// Tests for FindAvailablePort

func TestFindAvailablePort_Success(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, 8080)
	assert.LessOrEqual(t, port, 8180)
}

func TestFindAvailablePort_SinglePort(t *testing.T) {
	// Find an available port first
	availablePort, err := FindAvailablePort(9000, 9100)
	require.NoError(t, err)

	// Should work with single port range
	port, err := FindAvailablePort(availablePort, availablePort)

	require.NoError(t, err)
	assert.Equal(t, availablePort, port)
}

func TestFindAvailablePort_NoAvailablePorts(t *testing.T) {
	// Create listeners on a range of ports
	startPort, err := FindAvailablePort(9000, 9100)
	require.NoError(t, err)

	// Start a server on the only port in range
	server := NewCallbackServer(startPort, "test")
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Try to find a port in the same single-port range
	port, err := FindAvailablePort(startPort, startPort)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available port")
	assert.Equal(t, 0, port)
}

func TestFindAvailablePort_InvalidRange(t *testing.T) {
	// End port before start port
	port, err := FindAvailablePort(8180, 8080)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available port")
	assert.Equal(t, 0, port)
}

func TestFindAvailablePort_HighPortNumbers(t *testing.T) {
	// Test with high port numbers
	port, err := FindAvailablePort(50000, 50100)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, 50000)
	assert.LessOrEqual(t, port, 50100)
}

func TestFindAvailablePort_ConsecutiveCalls(t *testing.T) {
	// Multiple calls should find different ports if previous ones are in use
	port1, err := FindAvailablePort(10000, 10100)
	require.NoError(t, err)

	// Start server on first port
	server1 := NewCallbackServer(port1, "test1")
	err = server1.Start()
	require.NoError(t, err)
	defer server1.Stop()

	// Find another port
	port2, err := FindAvailablePort(10000, 10100)
	require.NoError(t, err)

	// Should be different (highly likely in range of 100 ports)
	assert.NotEqual(t, port1, port2)
}

// NOTE: OpenBrowser tests are skipped as they would actually open a browser.
// The function is platform-dependent and tested manually.

// Integration test for full OAuth callback flow

func TestCallbackServer_FullFlow(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "integration-test-state-abc123"
	expectedCode := "integration-auth-code-xyz789"

	server := NewCallbackServer(port, expectedState)

	// Start server
	err = server.Start()
	require.NoError(t, err)

	// Verify redirect URI
	redirectURI := server.RedirectURI()
	assert.Contains(t, redirectURI, fmt.Sprintf(":%d", port))
	assert.Contains(t, redirectURI, "/callback")

	// Simulate OAuth provider callback
	go func() {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("%s?code=%s&state=%s",
			redirectURI, expectedCode, expectedState))
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Wait for code
	code, err := server.WaitForCode(5 * time.Second)
	require.NoError(t, err)
	assert.Equal(t, expectedCode, code)

	// Stop server
	err = server.Stop()
	require.NoError(t, err)
}

func TestCallbackServer_MultipleStopCalls(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)

	// Multiple stop calls should not panic or error
	for i := 0; i < 3; i++ {
		err = server.Stop()
		require.NoError(t, err, "Stop call %d failed", i)
	}
}

func TestCallbackServer_StateValidation_CaseSensitive(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "TestState123"
	wrongState := "teststate123" // Different case

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request with wrong case
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=somecode&state=%s",
		port, wrongState))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should error due to case-sensitive comparison
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case err := <-server.errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state mismatch")
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestCallbackServer_HandleCallback_LongValues(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	// Create very long state and code values
	longState := string(make([]byte, 1000))
	for range longState {
		longState = "a" + longState[1:]
	}
	longCode := string(make([]byte, 1000))
	for range longCode {
		longCode = "b" + longCode[1:]
	}

	server := NewCallbackServer(port, longState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Make callback request with long values
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=%s&state=%s",
		port, url.QueryEscape(longCode), url.QueryEscape(longState)))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should handle long values correctly
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case code := <-server.codeChan:
		assert.Equal(t, longCode, code)
	case <-ctx.Done():
		t.Fatal("timeout waiting for code")
	}
}

func TestCallbackServer_HTTPMethodValidation(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	expectedState := "test-state"
	expectedCode := "test-code"

	server := NewCallbackServer(port, expectedState)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Test GET request (should work)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=%s&state=%s",
		port, expectedCode, expectedState))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify code was received
	select {
	case code := <-server.codeChan:
		assert.Equal(t, expectedCode, code)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for code")
	}

	// Test POST request (should also work - http.HandlerFunc handles all methods)
	formData := url.Values{}
	formData.Set("code", "post-code")
	formData.Set("state", expectedState)

	resp2, err := http.PostForm(fmt.Sprintf("http://localhost:%d/callback", port), formData)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

func TestCallbackServer_ServerTimeouts(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Verify server has timeout configured
	assert.NotNil(t, server.server)
	assert.Equal(t, 10*time.Second, server.server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.server.WriteTimeout)
}

func TestCallbackServer_InvalidPath(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Request to non-callback path should return 404
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/wrongpath", port))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCallbackServer_RootPath(t *testing.T) {
	port, err := FindAvailablePort(8080, 8180)
	require.NoError(t, err)

	server := NewCallbackServer(port, "test-state")

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Request to root path should return 404
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
