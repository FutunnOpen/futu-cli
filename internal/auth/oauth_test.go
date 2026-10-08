package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// ── TokenResponse parsing ───────────────────────────────────────────────────

func TestParseTokenResponse_Valid(t *testing.T) {
	body := `{
		"access_token": "at-abc",
		"refresh_token": "rt-xyz",
		"expires_in": 3600,
		"token_type": "bearer",
		"scope": "quote:read"
	}`
	resp, err := parseTokenResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "at-abc" {
		t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "at-abc")
	}
	if resp.RefreshToken != "rt-xyz" {
		t.Errorf("RefreshToken = %q, want %q", resp.RefreshToken, "rt-xyz")
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, 3600)
	}
	if resp.Scope != "quote:read" {
		t.Errorf("Scope = %q, want %q", resp.Scope, "quote:read")
	}
}

func TestParseTokenResponse_InvalidJSON(t *testing.T) {
	_, err := parseTokenResponse([]byte(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── parseTokenResponseOrError ───────────────────────────────────────────────

func TestParseTokenResponseOrError_Success(t *testing.T) {
	body := `{"access_token":"tok","refresh_token":"ref","expires_in":100,"token_type":"bearer"}`
	resp, err := parseTokenResponseOrError([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "tok")
	}
}

func TestParseTokenResponseOrError_OAuthError(t *testing.T) {
	body := `{"error":"invalid_grant","error_description":"code expired"}`
	_, err := parseTokenResponseOrError([]byte(body))
	if err == nil {
		t.Fatal("expected error for invalid_grant")
	}
	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected *OAuthError, got %T", err)
	}
	if oauthErr.Code != "invalid_grant" {
		t.Errorf("Code = %q, want %q", oauthErr.Code, "invalid_grant")
	}
	if oauthErr.Description != "code expired" {
		t.Errorf("Description = %q, want %q", oauthErr.Description, "code expired")
	}
}

// ── OAuthError.Error() ─────────────────────────────────────────────────────

func TestOAuthError_Error(t *testing.T) {
	e := &OAuthError{Code: "test_code", Description: "test desc"}
	got := e.Error()
	want := "oauth error: test_code — test desc"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// ── BuildAuthorizationURL ───────────────────────────────────────────────────

func TestBuildAuthorizationURL(t *testing.T) {
	pkce := &PKCEParams{
		CodeVerifier:  "verifier-ignored-in-url",
		CodeChallenge: "challenge123",
		Method:        "S256",
	}
	authURL := BuildAuthorizationURL("cid-001", "http://127.0.0.1:18888/callback", pkce, "state-abc")

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	checks := map[string]string{
		"client_id":             "cid-001",
		"redirect_uri":          "http://127.0.0.1:18888/callback",
		"code_challenge":        "challenge123",
		"code_challenge_method": "S256",
		"response_type":         "code",
		"scope":                 defaultScopes,
		"state":                 "state-abc",
	}
	for key, want := range checks {
		if got := parsed.Query().Get(key); got != want {
			t.Errorf("param %s = %q, want %q", key, got, want)
		}
	}

	if !strings.HasSuffix(parsed.Path, authorizeURL) {
		t.Errorf("path = %q, want suffix %q", parsed.Path, authorizeURL)
	}
}

// ── GenerateState ───────────────────────────────────────────────────────────

func TestGenerateState_Length(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedLen := stateBytes * 2
	if len(state) != expectedLen {
		t.Errorf("state length = %d, want %d", len(state), expectedLen)
	}
}

func TestGenerateState_Unique(t *testing.T) {
	s1, _ := GenerateState()
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("two states should not be identical")
	}
}

// ── BuildClientName ─────────────────────────────────────────────────────────

func TestBuildClientName_WithOverride(t *testing.T) {
	name := BuildClientName("MyApp")
	if name != "MyApp (Futu CLI)" {
		t.Errorf("got %q, want %q", name, "MyApp (Futu CLI)")
	}
}

func TestBuildClientName_Default(t *testing.T) {
	name := BuildClientName("")
	if len(name) == 0 {
		t.Error("expected non-empty client name")
	}
	suffix := " (Futu CLI)"
	if name[len(name)-len(suffix):] != suffix {
		t.Errorf("client name %q should end with %q", name, suffix)
	}
}

// ── HTTP-level integration tests with httptest ──────────────────────────────

func TestPostJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	body, err := postJSON(srv.URL+"/test", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %q, want %q", result["status"], "ok")
	}
}

func TestPostJSON_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad_request"}`))
	}))
	defer srv.Close()

	_, err := postJSON(srv.URL+"/test", map[string]string{})
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

func TestPostJSON_ServerDown(t *testing.T) {
	_, err := postJSON("http://127.0.0.1:1/unreachable", map[string]string{})
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

// ── postForm ────────────────────────────────────────────────────────────────

func TestPostForm_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type = %q, want form-urlencoded", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q, want %q", got, "authorization_code")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
	}))
	defer srv.Close()

	data := url.Values{"grant_type": {"authorization_code"}, "code": {"abc"}}
	body, err := postForm(srv.URL+"/token", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["access_token"] != "tok" {
		t.Errorf("access_token = %q, want %q", result["access_token"], "tok")
	}
}

func TestPostForm_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer srv.Close()

	_, err := postForm(srv.URL+"/token", url.Values{})
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
}

func TestOAuthHTTPClientTimeout(t *testing.T) {
	if oauthHTTPClient.Timeout != 30*time.Second {
		t.Fatalf("OAuth HTTP timeout = %s, want %s", oauthHTTPClient.Timeout, 30*time.Second)
	}
}

// ── readResponseBody ────────────────────────────────────────────────────────

func TestReadResponseBody_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`hello`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := readResponseBody(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}
}

func TestReadResponseBody_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	_, err = readResponseBody(resp)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// ── parseRegistrationResponse ───────────────────────────────────────────────

func TestParseRegistrationResponse_Valid(t *testing.T) {
	body := `{
		"client_id": "cid-001",
		"client_id_issued_at": 1700000000,
		"registration_access_token": "rat-xyz",
		"client_name": "test (Futu CLI)",
		"redirect_uris": ["http://127.0.0.1:18888/callback"],
		"scope": "quote:read",
		"pkce_required": true
	}`
	resp, err := parseRegistrationResponse([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ClientID != "cid-001" {
		t.Errorf("ClientID = %q, want %q", resp.ClientID, "cid-001")
	}
	if !resp.PKCERequired {
		t.Error("PKCERequired should be true")
	}
	if len(resp.RedirectURIs) != 1 || resp.RedirectURIs[0] != "http://127.0.0.1:18888/callback" {
		t.Errorf("RedirectURIs = %v", resp.RedirectURIs)
	}
}

func TestParseRegistrationResponse_InvalidJSON(t *testing.T) {
	_, err := parseRegistrationResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
