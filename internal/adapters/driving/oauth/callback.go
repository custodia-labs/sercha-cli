// Package oauth provides OAuth callback server and browser utilities.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// CallbackServer handles OAuth redirect callbacks.
// It starts a local HTTP server to receive the authorization code.
type CallbackServer struct {
	mu            sync.Mutex
	port          int
	expectedState string
	codeChan      chan string
	errChan       chan error
	server        *http.Server
	listener      net.Listener
}

// NewCallbackServer creates a new OAuth callback server.
// The expectedState is used to validate the callback matches the request.
func NewCallbackServer(port int, expectedState string) *CallbackServer {
	return &CallbackServer{
		port:          port,
		expectedState: expectedState,
		codeChan:      make(chan string, 1),
		errChan:       make(chan error, 1),
	}
}

// Start starts the callback server on the configured port.
// If port is 0, a random available port will be chosen.
func (s *CallbackServer) Start() error {
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

	// Store the actual port (important when port was 0)
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		s.port = tcpAddr.Port
	}

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
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.errChan <- fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, successHTML(fmt.Sprintf("Authorization failed: %s", html.EscapeString(errDesc)), ""))
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
	fmt.Fprint(w, successHTML("Authorization successful!", "You can close this window and return to the application."))
}

// WaitForCode blocks until the authorization code is received or timeout.
func (s *CallbackServer) WaitForCode(timeout time.Duration) (string, error) {
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
func (s *CallbackServer) Stop() error {
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
func (s *CallbackServer) Port() int {
	return s.port
}

// RedirectURI returns the redirect URI for this callback server.
func (s *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://localhost:%d/callback", s.port)
}

//nolint:misspell,lll // CSS properties use American spelling, SVG paths exceed line length
func successHTML(title, message string) string {
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
            background: #FAFAFA;
        }
        .container {
            text-align: center;
            background: white;
            padding: 48px 64px;
            border-radius: 16px;
            border: 1px solid #C7C8CC;
            box-shadow: 0 4px 24px rgba(0,0,0,0.08);
        }
        .logo {
            width: 200px;
            height: auto;
            margin-bottom: 32px;
        }
        h1 {
            color: #333F50;
            margin: 0 0 8px 0;
            font-size: 24px;
            font-weight: 600;
        }
        p {
            color: #7B8088;
            margin: 0;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="container">
        <svg class="logo" viewBox="540 440 850 200" xmlns="http://www.w3.org/2000/svg">
            <g id="name">
                <path fill="#333F50" d="M625.87,527.17c-19.7-3.51-34.82-7.29-34.82-21.86c0-11.07,10.26-18.09,25.1-18.09c18.35,0,30.23,8.1,36.17,19.71l32.93-16.47c-11.61-25.91-37.52-39.68-67.48-39.68c-37.52,0-66.13,24.29-66.13,56.15c0,41.03,36.98,50.75,64.24,55.33c18.89,3.24,33.2,5.67,33.2,20.79c0,12.96-12.96,19.97-29.96,19.97c-18.9,0-31.58-6.75-37.52-19.97l-32.93,16.47c11.61,25.91,37.79,39.68,70.45,39.68c38.6,0,69.37-23.75,69.37-56.41C688.49,541.48,652.86,531.76,625.87,527.17z"/>
                <path fill="#333F50" d="M765.06,497.21c-41.3,0-69.37,31.85-69.37,70.45c0,42.65,31.31,71.53,72.88,71.53c28.34,0,50.48-13.77,62.08-37.79l-29.42-14.85c-6.21,12.42-18.35,19.43-32.93,19.43c-19.17,0-32.93-11.88-35.63-29.69H832v-11.34C832,528.52,807.44,497.21,765.06,497.21z M734.28,554.97c2.7-13.5,13.77-24.29,30.5-24.29c16.2,0,27.53,10.53,29.15,24.29H734.28z"/>
                <path fill="#333F50" d="M839.75,559.02v76.66h38.33v-75.31c0-15.93,9.18-25.64,24.29-25.64c7.02,0,14.85,1.89,21.32,5.67v-37.25c-6.75-2.97-16.2-4.59-24.83-4.59C859.18,498.56,839.75,524.74,839.75,559.02z"/>
                <path fill="#333F50" d="M1155.59,498.29c-14.58,0-27.8,4.59-37.79,11.88v-69.64h-38.33v195.16h38.33v-65.05c0-21.05,12.69-36.17,31.85-36.17c19.43,0,29.69,16.74,29.69,35.63v65.32l38.33,0.27v-69.37C1217.67,524.2,1196.35,498.29,1155.59,498.29z"/>
                <path fill="#333F50" d="M1298.72,498.02c-41.03,0-72.34,32.12-72.34,70.45c0,38.87,27.26,70.72,66.13,70.72c18.36,0,32.93-7.29,42.38-19.16v15.66h36.44v-67.21C1371.33,528.25,1340.56,498.02,1298.72,498.02z M1298.72,603.29c-19.7,0-34.01-15.66-34.01-34.55c0-18.9,14.31-34.55,34.01-34.55c19.7,0,34.28,15.66,34.28,34.55C1333,587.63,1318.43,603.29,1298.72,603.29z"/>
            </g>
            <g id="icon">
                <path fill="#6675FF" d="M1003.81,544.99c7.19,1.01,13.3,5.33,16.77,11.38h8.34c-4.15-10.25-13.69-17.73-25.11-18.93V544.99z"/>
                <path fill="#6675FF" d="M1070.17,587.54l-32.86,0c-3.38,6.58-8.47,12.13-14.69,16.07c-0.04,0.02-0.07,0.04-0.11,0.06l6.29,11.87l5.51-2.92l7.6,14.35c0.99-0.65,2-1.32,3.02-2.04C1056.94,615.59,1065.92,602.56,1070.17,587.54z"/>
                <path fill="#6675FF" d="M1018.54,620.97l-6.68-12.61c-0.51,0.14-1.01,0.31-1.53,0.44c-2.63,0.64-5.35,1.02-8.14,1.13c-0.53,0.02-1.07,0.06-1.61,0.06c-22.78,0-41.25-18.47-41.25-41.25s18.47-41.25,41.25-41.25c16.74,0,31.14,9.98,37.61,24.3h32.91c-7.3-32.22-36.09-56.28-70.51-56.28c-39.94,0-72.32,32.38-72.32,72.32c0,39.94,32.38,72.32,72.32,72.32c3.92,0,7.77-0.32,11.52-0.92c0,0,3-0.23,8.09-1.78l-7.17-13.54L1018.54,620.97z"/>
            </g>
        </svg>
        <h1>%s</h1>
        <p>%s</p>
    </div>
</body>
</html>`, title, message)
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

// GenerateCodeVerifier generates a random PKCE code verifier.
func GenerateCodeVerifier() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure but still usable random
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateCodeChallenge generates a PKCE code challenge from a verifier.
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
