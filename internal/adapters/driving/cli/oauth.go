package cli

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// OAuthCallbackServer handles OAuth redirect callbacks.
// It starts a local HTTP server to receive the authorization code.
type OAuthCallbackServer struct {
	mu            sync.Mutex
	port          int
	expectedState string
	codeChan      chan string
	errChan       chan error
	server        *http.Server
	listener      net.Listener
}

// NewOAuthCallbackServer creates a new OAuth callback server.
// The expectedState is used to validate the callback matches the request.
func NewOAuthCallbackServer(port int, expectedState string) *OAuthCallbackServer {
	return &OAuthCallbackServer{
		port:          port,
		expectedState: expectedState,
		codeChan:      make(chan string, 1),
		errChan:       make(chan error, 1),
	}
}

// Start starts the callback server on the configured port.
func (s *OAuthCallbackServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case s.errChan <- err:
			default:
			}
		}
	}()

	return nil
}

// handleCallback processes the OAuth callback request.
func (s *OAuthCallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.errChan <- fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, successHTML(fmt.Sprintf("Authorization failed: %s", errDesc), ""))
		return
	}

	// Validate state parameter
	state := r.URL.Query().Get("state")
	if state != s.expectedState {
		s.errChan <- fmt.Errorf("state mismatch: expected %s, got %s", s.expectedState, state)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML("Authorization failed: invalid state parameter", ""))
		return
	}

	// Extract authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		s.errChan <- fmt.Errorf("no authorization code received")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML("Authorization failed: no code received", ""))
		return
	}

	// Send code to channel
	select {
	case s.codeChan <- code:
	default:
	}

	// Return success page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, successHTML("Authorization successful!", "You can close this window and return to the CLI."))
}

// WaitForCode blocks until the authorization code is received or timeout.
func (s *OAuthCallbackServer) WaitForCode(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case code := <-s.codeChan:
		return code, nil
	case err := <-s.errChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("timeout waiting for authorization callback")
	}
}

// Stop shuts down the callback server.
func (s *OAuthCallbackServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Port returns the port the server is listening on.
func (s *OAuthCallbackServer) Port() int {
	return s.port
}

// RedirectURI returns the redirect URI for this callback server.
func (s *OAuthCallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://localhost:%d/callback", s.port)
}

//nolint:misspell // CSS properties use American spelling (center, color)
func successHTML(title, message string) string {
	escapedTitle := html.EscapeString(title)
	escapedMessage := html.EscapeString(message)
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Sercha - OAuth Callback</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            text-align: center;
            background: white;
            padding: 40px 60px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        }
        h1 { color: #333; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p>%s</p>
    </div>
</body>
</html>`, escapedTitle, escapedMessage)
}

// OpenBrowser opens the default browser to the given URL.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// FindAvailablePort finds an available port in the given range.
func FindAvailablePort(startPort, endPort int) (int, error) {
	for port := startPort; port <= endPort; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", startPort, endPort)
}
