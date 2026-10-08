package auth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	callbackPath    = "/callback"
	callbackAddr    = "127.0.0.1"
	callbackTimeout = 120 * time.Second
)

const callbackPortStart = 18888
const callbackPortRetries = 3

const callbackSuccessHTML = `<!DOCTYPE html>
<html><body style="font-family:sans-serif;text-align:center;padding:60px">
<h2>授权成功</h2><p>请关闭此页面，返回终端继续操作。</p>
</body></html>`

const callbackErrorHTML = `<!DOCTYPE html>
<html><body style="font-family:sans-serif;text-align:center;padding:60px">
<h2>授权失败</h2><p>%s</p>
</body></html>`

type authorizationTimeoutError struct {
	cause error
}

func (e authorizationTimeoutError) Error() string {
	return "authorization timed out"
}

func (e authorizationTimeoutError) Unwrap() error {
	return e.cause
}

// CallbackResult holds the authorization code received from the OAuth redirect.
type CallbackResult struct {
	Code  string
	State string
	Error string
}

// ParseCallbackURL extracts OAuth2 parameters from a manually pasted callback.
func ParseCallbackURL(callbackURL, expectedRedirectURI string) (*CallbackResult, error) {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return nil, fmt.Errorf("parse callback URL: %w", err)
	}
	if len(parsed.Scheme) == 0 || len(parsed.Host) == 0 {
		return nil, fmt.Errorf("callback URL must be an absolute URL")
	}
	if !sameRedirectTarget(parsed, expectedRedirectURI) {
		return nil, fmt.Errorf("callback URL does not match the expected redirect URI")
	}
	result := callbackResultFromValues(parsed.Query())
	if len(result.Error) == 0 && len(strings.TrimSpace(result.Code)) == 0 {
		return nil, fmt.Errorf("callback URL is missing authorization code")
	}
	return result, nil
}

func callbackResultFromValues(query url.Values) *CallbackResult {
	return &CallbackResult{
		Code: query.Get("code"), State: query.Get("state"), Error: query.Get("error"),
	}
}

func sameRedirectTarget(callbackURL *url.URL, expectedRedirectURI string) bool {
	expected, err := url.Parse(expectedRedirectURI)
	if err != nil {
		return false
	}
	return callbackURL.Scheme == expected.Scheme &&
		callbackURL.Host == expected.Host &&
		callbackURL.Path == expected.Path
}

// CallbackServer is a localhost HTTP server that captures the OAuth authorization code.
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
	result   chan CallbackResult
}

// NewCallbackServer creates and starts a callback server listening on a local port.
// It tries ports 18888–18890 until one is available.
func NewCallbackServer() (*CallbackServer, error) {
	listener, err := listenOnAvailablePort()
	if err != nil {
		return nil, err
	}
	return startCallbackServer(listener), nil
}

// NewCallbackServerForRedirectURI listens on a previously registered callback URI.
func NewCallbackServerForRedirectURI(redirectURI string) (*CallbackServer, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse registered redirect URI: %w", err)
	}
	if parsed.Scheme != "http" || parsed.Hostname() != callbackAddr || parsed.Path != callbackPath {
		return nil, fmt.Errorf("unsupported registered redirect URI: %s", redirectURI)
	}
	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		return nil, fmt.Errorf("registered callback address %s is unavailable: %w", parsed.Host, err)
	}
	return startCallbackServer(listener), nil
}

func startCallbackServer(listener net.Listener) *CallbackServer {
	cs := &CallbackServer{
		listener: listener,
		result:   make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, cs.handleCallback)
	cs.server = &http.Server{Handler: mux}

	go func() { _ = cs.server.Serve(listener) }()
	return cs
}

// Port returns the port the callback server is listening on.
func (s *CallbackServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// RedirectURI returns the full redirect URI for OAuth registration and authorization.
func (s *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://%s:%d%s", callbackAddr, s.Port(), callbackPath)
}

// WaitForCallback blocks until the authorization callback arrives or the context expires.
func (s *CallbackServer) WaitForCallback(ctx context.Context) (*CallbackResult, error) {
	select {
	case result := <-s.result:
		s.shutdown()
		return &result, nil
	case <-ctx.Done():
		s.shutdown()
		return nil, authorizationTimeoutError{cause: ctx.Err()}
	}
}

// Shutdown stops the callback server without waiting for a callback.
func (s *CallbackServer) Shutdown() {
	s.shutdown()
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	result := CallbackResult{
		Code:  query.Get("code"),
		State: query.Get("state"),
		Error: query.Get("error"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(result.Error) > 0 {
		fmt.Fprintf(w, callbackErrorHTML, html.EscapeString(result.Error))
	} else if len(result.Code) > 0 {
		fmt.Fprint(w, callbackSuccessHTML)
	} else {
		result.Error = "missing authorization code"
		fmt.Fprintf(w, callbackErrorHTML, html.EscapeString(result.Error))
	}

	select {
	case s.result <- result:
	default:
	}
}

func (s *CallbackServer) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func listenOnAvailablePort() (net.Listener, error) {
	for i := 0; i < callbackPortRetries; i++ {
		port := callbackPortStart + i
		addr := fmt.Sprintf("%s:%d", callbackAddr, port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
	}
	return nil, fmt.Errorf("unable to listen on ports %d–%d; please free one of these ports",
		callbackPortStart, callbackPortStart+callbackPortRetries-1)
}
