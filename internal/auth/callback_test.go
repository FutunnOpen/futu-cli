package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewCallbackServer_StartsAndListens(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer srv.Shutdown()

	port := srv.Port()
	if port < callbackPortStart || port > callbackPortStart+callbackPortRetries-1 {
		t.Errorf("port = %d, want in range [%d, %d]", port, callbackPortStart, callbackPortStart+callbackPortRetries-1)
	}
}

func TestCallbackServer_RedirectURI(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Shutdown()

	uri := srv.RedirectURI()
	if !strings.HasPrefix(uri, "http://127.0.0.1:") {
		t.Errorf("RedirectURI = %q, want http://127.0.0.1:...", uri)
	}
	if !strings.HasSuffix(uri, callbackPath) {
		t.Errorf("RedirectURI = %q, want suffix %q", uri, callbackPath)
	}
}

func TestCallbackServerForRedirectURI(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	server, err := NewCallbackServerForRedirectURI(redirectURI)
	if err != nil {
		t.Fatalf("NewCallbackServerForRedirectURI() error = %v", err)
	}
	defer server.Shutdown()
	if server.RedirectURI() != redirectURI {
		t.Fatalf("RedirectURI() = %q, want %q", server.RedirectURI(), redirectURI)
	}
}

func TestCallbackServerForRedirectURIRejectsNonLoopback(t *testing.T) {
	_, err := NewCallbackServerForRedirectURI("https://example.com/callback")
	if err == nil {
		t.Fatal("expected unsupported redirect URI error")
	}
}

func TestCallbackServer_SuccessfulCallback(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s?code=test-code&state=test-state", srv.Port(), callbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback GET failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Code != "test-code" {
		t.Errorf("Code = %q, want %q", result.Code, "test-code")
	}
	if result.State != "test-state" {
		t.Errorf("State = %q, want %q", result.State, "test-state")
	}
	if len(result.Error) > 0 {
		t.Errorf("Error = %q, want empty", result.Error)
	}
}

func TestCallbackServer_ErrorCallback(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s?error=access_denied", srv.Port(), callbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback GET failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Error != "access_denied" {
		t.Errorf("Error = %q, want %q", result.Error, "access_denied")
	}
}

func TestCallbackServer_MissingCode(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d%s?state=s", srv.Port(), callbackPath)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("callback GET failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Error != "missing authorization code" {
		t.Errorf("Error = %q, want %q", result.Error, "missing authorization code")
	}
}

func TestCallbackServer_Timeout(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = srv.WaitForCallback(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want to contain 'timed out'", err.Error())
	}
	if strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("error = %q, should not expose context deadline", err.Error())
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error should wrap context deadline exceeded")
	}
}

func TestParseCallbackURL(t *testing.T) {
	result, err := ParseCallbackURL(
		"http://127.0.0.1:18888/callback?code=code-1&state=state-1",
		"http://127.0.0.1:18888/callback",
	)
	if err != nil {
		t.Fatalf("ParseCallbackURL: %v", err)
	}
	if result.Code != "code-1" || result.State != "state-1" {
		t.Fatalf("callback result = %#v", result)
	}
}

func TestParseCallbackURLRejectsDifferentRedirect(t *testing.T) {
	_, err := ParseCallbackURL(
		"http://127.0.0.1:18889/callback?code=code-1",
		"http://127.0.0.1:18888/callback",
	)
	if err == nil {
		t.Fatal("expected an error for a different redirect target")
	}
}
