package auth

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeleteTokenRemovesFileAndIsIdempotent(t *testing.T) {
	previousDelete := deleteLegacyToken
	legacyDeleteCalls := 0
	deleteLegacyToken = func() error {
		legacyDeleteCalls++
		return nil
	}
	t.Cleanup(func() { deleteLegacyToken = previousDelete })
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte("token"), tokenFilePermissions); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if err := DeleteToken(path); err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("token file still exists: %v", err)
	}
	if err := DeleteToken(path); err != nil {
		t.Fatalf("second DeleteToken() error = %v", err)
	}
	if legacyDeleteCalls != 2 {
		t.Fatalf("legacy keyring delete calls = %d, want 2", legacyDeleteCalls)
	}
}

// ── TokenStore.IsExpired ────────────────────────────────────────────────────

func TestTokenStore_IsExpired_FreshToken(t *testing.T) {
	store := &TokenStore{
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if store.IsExpired() {
		t.Error("token with 10 minutes remaining should not be expired")
	}
}

func TestTokenStore_IsExpired_WithinBuffer(t *testing.T) {
	store := &TokenStore{
		ExpiresAt: time.Now().Add(20 * time.Second),
	}
	if !store.IsExpired() {
		t.Error("token within 30-second buffer should be considered expired")
	}
}

func TestTokenStore_IsExpired_AlreadyExpired(t *testing.T) {
	store := &TokenStore{
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	if !store.IsExpired() {
		t.Error("token expired 5 minutes ago should be expired")
	}
}

func TestTokenStore_IsExpired_ExactlyAtBuffer(t *testing.T) {
	store := &TokenStore{
		ExpiresAt: time.Now().Add(tokenExpiryBuffer),
	}
	// At exactly the buffer boundary, time.Now().After(expiresAt - buffer) == true
	// because time.Now() >= expiresAt - buffer.
	if !store.IsExpired() {
		t.Error("token at exactly the buffer boundary should be considered expired")
	}
}

// ── unmarshalToken ──────────────────────────────────────────────────────────

func TestUnmarshalToken_Valid(t *testing.T) {
	data := []byte(`{
		"access_token": "at-123",
		"refresh_token": "rt-456",
		"expires_at": "2026-12-31T23:59:59Z",
		"client_name": "test (Futu CLI)"
	}`)
	store, err := unmarshalToken(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.AccessToken != "at-123" {
		t.Errorf("AccessToken = %q, want %q", store.AccessToken, "at-123")
	}
	if store.RefreshToken != "rt-456" {
		t.Errorf("RefreshToken = %q, want %q", store.RefreshToken, "rt-456")
	}
	if store.ClientName != "test (Futu CLI)" {
		t.Errorf("ClientName = %q, want %q", store.ClientName, "test (Futu CLI)")
	}
}

func TestUnmarshalToken_InvalidJSON(t *testing.T) {
	_, err := unmarshalToken([]byte(`{broken`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── buildUpdatedStore ───────────────────────────────────────────────────────

func TestBuildUpdatedStore(t *testing.T) {
	resp := &TokenResponse{
		AccessToken:  "new-at",
		RefreshToken: "new-rt",
		ExpiresIn:    7200,
	}
	before := time.Now()
	previous := &TokenStore{ClientName: "my-client"}
	store := buildUpdatedStore(resp, previous, "cid-001")
	after := time.Now()

	if store.AccessToken != "new-at" {
		t.Errorf("AccessToken = %q, want %q", store.AccessToken, "new-at")
	}
	if store.RefreshToken != "new-rt" {
		t.Errorf("RefreshToken = %q, want %q", store.RefreshToken, "new-rt")
	}
	if store.ClientName != "my-client" {
		t.Errorf("ClientName = %q, want %q", store.ClientName, "my-client")
	}
	if store.ClientID != "cid-001" {
		t.Errorf("ClientID = %q, want %q", store.ClientID, "cid-001")
	}

	expectedMin := before.Add(7200 * time.Second)
	expectedMax := after.Add(7200 * time.Second)
	if store.ExpiresAt.Before(expectedMin) || store.ExpiresAt.After(expectedMax) {
		t.Errorf("ExpiresAt = %v, expected between %v and %v", store.ExpiresAt, expectedMin, expectedMax)
	}
}

func TestEnsureValidStoreRefreshesExpiredToken(t *testing.T) {
	previousClient := oauthHTTPClient
	oauthHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != oauthBaseURL+tokenURL {
			t.Fatalf("refresh URL = %q, want %q", req.URL.String(), oauthBaseURL+tokenURL)
		}
		if err := req.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := req.PostForm.Get("grant_type"); got != grantTypeRefresh {
			t.Fatalf("grant_type = %q, want %q", got, grantTypeRefresh)
		}
		if got := req.PostForm.Get("client_id"); got != "stored-client-id" {
			t.Fatalf("client_id = %q, want stored-client-id", got)
		}
		if got := req.PostForm.Get("refresh_token"); got != "old-refresh" {
			t.Fatalf("refresh_token = %q, want old-refresh", got)
		}
		body := `{"access_token":"new-access","expires_in":3600,"token_type":"bearer"}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	t.Cleanup(func() { oauthHTTPClient = previousClient })

	path := filepath.Join(t.TempDir(), "token.json")
	store := &TokenStore{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
		ClientID:     "stored-client-id",
		ClientName:   "test-client",
	}
	if err := store.Save(path); err != nil {
		t.Fatalf("save token: %v", err)
	}

	refreshed, err := EnsureValidStore("", path)
	if err != nil {
		t.Fatalf("EnsureValidStore() error = %v", err)
	}
	if refreshed.AccessToken != "new-access" {
		t.Fatalf("AccessToken = %q, want new-access", refreshed.AccessToken)
	}
	if refreshed.RefreshToken != "old-refresh" {
		t.Fatalf("RefreshToken = %q, want old-refresh", refreshed.RefreshToken)
	}
	saved, err := LoadToken(path)
	if err != nil {
		t.Fatalf("load saved token: %v", err)
	}
	if saved.AccessToken != "new-access" || saved.RefreshToken != "old-refresh" {
		t.Fatalf("saved token = %#v", saved)
	}
}

// ── removeFileIfExists ──────────────────────────────────────────────────────

func TestRemoveFileIfExists_NonExistentFile(t *testing.T) {
	err := removeFileIfExists("/tmp/futu-cli-test-nonexistent-file-12345")
	if err != nil {
		t.Errorf("expected nil for non-existent file, got: %v", err)
	}
}
