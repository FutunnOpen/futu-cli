package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/futunn/futu/internal/auth"
	"github.com/futunn/futu/internal/config"
	"github.com/zalando/go-keyring"
)

func TestEnsureClientIDReusesStoredRegistration(t *testing.T) {
	previousConfig := cfg
	cfg = &config.Config{
		APIBase:           "https://webapi.futunn.com",
		ClientID:          "stored-client-id",
		ClientIDAPIBase:   "webapi.futunn.com",
		ClientIDOAuthBase: "https://webapi.futunn.com",
		RedirectURI:       "http://127.0.0.1:18888/callback",
	}
	t.Cleanup(func() { cfg = previousConfig })

	clientID, err := ensureClientID([]string{cfg.RedirectURI})
	if err != nil {
		t.Fatalf("ensureClientID() error = %v", err)
	}
	if clientID != cfg.ClientID {
		t.Fatalf("client ID = %q, want %q", clientID, cfg.ClientID)
	}
}

func TestEnsureClientIDRegistersWhenAPIBaseHostChanges(t *testing.T) {
	previousConfig := cfg
	previousRegister := registerOAuthClient
	t.Setenv("HOME", t.TempDir())
	cfg = &config.Config{
		APIBase:         "https://alt-webapi.example.test",
		ClientID:        "old-client-id",
		ClientIDAPIBase: "webapi.futunn.com",
		RedirectURI:     "http://127.0.0.1:18888/callback",
		TokenFile:       t.TempDir() + "/token.json",
	}
	registerOAuthClient = func(baseURL, _ string, redirectURIs []string) (*auth.RegistrationResponse, error) {
		if baseURL != "https://alt-webapi.example.test" {
			t.Fatalf("baseURL = %q, want https://alt-webapi.example.test", baseURL)
		}
		if len(redirectURIs) != 1 || redirectURIs[0] != cfg.RedirectURI {
			t.Fatalf("redirect URIs = %#v", redirectURIs)
		}
		return &auth.RegistrationResponse{ClientID: "new-client-id"}, nil
	}
	t.Cleanup(func() {
		cfg = previousConfig
		registerOAuthClient = previousRegister
	})

	clientID, err := ensureClientID([]string{cfg.RedirectURI})
	if err != nil {
		t.Fatalf("ensureClientID() error = %v", err)
	}
	if clientID != "new-client-id" || cfg.ClientID != "new-client-id" {
		t.Fatalf("client ID = %q, config = %q; want new-client-id", clientID, cfg.ClientID)
	}
	if cfg.ClientIDAPIBase != "alt-webapi.example.test" {
		t.Fatalf("client_id_api_base = %q", cfg.ClientIDAPIBase)
	}
	if cfg.ClientIDOAuthBase != "https://alt-webapi.example.test" {
		t.Fatalf("client_id_oauth_base = %q", cfg.ClientIDOAuthBase)
	}
}

func TestEnsureClientIDRegistersWhenAPIBaseHostIsMissing(t *testing.T) {
	previousConfig := cfg
	previousRegister := registerOAuthClient
	t.Setenv("HOME", t.TempDir())
	cfg = &config.Config{
		APIBase:     "https://webapi.futunn.com",
		ClientID:    "old-client-id",
		RedirectURI: "http://127.0.0.1:18888/callback",
		TokenFile:   t.TempDir() + "/token.json",
	}
	registerOAuthClient = func(baseURL, _ string, redirectURIs []string) (*auth.RegistrationResponse, error) {
		if baseURL != "https://webapi.futunn.com" {
			t.Fatalf("baseURL = %q, want https://webapi.futunn.com", baseURL)
		}
		if len(redirectURIs) != 1 || redirectURIs[0] != cfg.RedirectURI {
			t.Fatalf("redirect URIs = %#v", redirectURIs)
		}
		return &auth.RegistrationResponse{ClientID: "new-client-id"}, nil
	}
	t.Cleanup(func() {
		cfg = previousConfig
		registerOAuthClient = previousRegister
	})

	clientID, err := ensureClientID([]string{cfg.RedirectURI})
	if err != nil {
		t.Fatalf("ensureClientID() error = %v", err)
	}
	if clientID != "new-client-id" || cfg.ClientID != "new-client-id" {
		t.Fatalf("client ID = %q, config = %q; want new-client-id", clientID, cfg.ClientID)
	}
	if cfg.ClientIDAPIBase != "webapi.futunn.com" {
		t.Fatalf("client_id_api_base = %q", cfg.ClientIDAPIBase)
	}
	if cfg.ClientIDOAuthBase != "https://webapi.futunn.com" {
		t.Fatalf("client_id_oauth_base = %q", cfg.ClientIDOAuthBase)
	}
}

func TestEnsureClientIDRegistersWhenOAuthBaseBindingIsMissing(t *testing.T) {
	previousConfig := cfg
	previousRegister := registerOAuthClient
	t.Setenv("HOME", t.TempDir())
	cfg = &config.Config{
		APIBase:         "https://alt-webapi.example.test",
		ClientID:        "old-client-id",
		ClientIDAPIBase: "alt-webapi.example.test",
		RedirectURI:     "http://127.0.0.1:18888/callback",
		TokenFile:       t.TempDir() + "/token.json",
	}
	registerOAuthClient = func(baseURL, _ string, redirectURIs []string) (*auth.RegistrationResponse, error) {
		if baseURL != "https://alt-webapi.example.test" {
			t.Fatalf("baseURL = %q, want https://alt-webapi.example.test", baseURL)
		}
		if len(redirectURIs) != 1 || redirectURIs[0] != cfg.RedirectURI {
			t.Fatalf("redirect URIs = %#v", redirectURIs)
		}
		return &auth.RegistrationResponse{ClientID: "new-client-id"}, nil
	}
	t.Cleanup(func() {
		cfg = previousConfig
		registerOAuthClient = previousRegister
	})

	clientID, err := ensureClientID([]string{cfg.RedirectURI})
	if err != nil {
		t.Fatalf("ensureClientID() error = %v", err)
	}
	if clientID != "new-client-id" || cfg.ClientID != "new-client-id" {
		t.Fatalf("client ID = %q, config = %q; want new-client-id", clientID, cfg.ClientID)
	}
	if cfg.ClientIDOAuthBase != "https://alt-webapi.example.test" {
		t.Fatalf("client_id_oauth_base = %q", cfg.ClientIDOAuthBase)
	}
}

func TestAPIBaseHostNormalizesSchemePortAndPath(t *testing.T) {
	got, err := apiBaseHost("https://WebAPI.Futunn.com:443/api/v1.0")
	if err != nil {
		t.Fatalf("apiBaseHost() error = %v", err)
	}
	if got != "webapi.futunn.com" {
		t.Fatalf("host = %q, want webapi.futunn.com", got)
	}
}

func TestAuthorizationURLUsesConfiguredAPIBase(t *testing.T) {
	previousConfig := cfg
	cfg = &config.Config{APIBase: "https://alt-webapi.example.test/api/v1.0"}
	t.Cleanup(func() { cfg = previousConfig })

	request := &authorizationRequest{
		ClientID:    "cid-001",
		RedirectURI: "http://127.0.0.1:18888/callback",
		State:       "state-abc",
		PKCE:        &auth.PKCEParams{CodeChallenge: "challenge123", Method: "S256"},
	}
	if err := assignAuthorizationURL(request); err != nil {
		t.Fatalf("assignAuthorizationURL() error = %v", err)
	}
	if !strings.HasPrefix(request.URL, "https://alt-webapi.example.test/oauth2/authorize/confirm?") {
		t.Fatalf("authorization URL = %q", request.URL)
	}
}

func TestStatusRefreshClientIDUsesMatchingConfiguredClientID(t *testing.T) {
	previousConfig := cfg
	cfg = &config.Config{
		APIBase:           "https://webapi.futunn.com",
		ClientID:          "configured-client-id",
		ClientIDAPIBase:   "webapi.futunn.com",
		ClientIDOAuthBase: "https://webapi.futunn.com",
	}
	t.Cleanup(func() { cfg = previousConfig })

	clientID, err := statusRefreshClientID(&auth.TokenStore{ClientID: "stored-client-id"})
	if err != nil {
		t.Fatalf("statusRefreshClientID() error = %v", err)
	}
	if clientID != cfg.ClientID {
		t.Fatalf("client ID = %q, want %q", clientID, cfg.ClientID)
	}
}

func TestStatusRefreshClientIDRejectsUnboundStoredClientID(t *testing.T) {
	previousConfig := cfg
	cfg = &config.Config{APIBase: "https://webapi.futunn.com"}
	t.Cleanup(func() { cfg = previousConfig })

	_, err := statusRefreshClientID(&auth.TokenStore{ClientID: "stored-client-id"})
	if err == nil {
		t.Fatal("expected stored token client_id without matching api_base to be rejected")
	}
}

func TestNewAuthStatusValidToken(t *testing.T) {
	now := time.Now()
	store := &auth.TokenStore{
		LoggedInAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
		ClientName: "test-client",
		Account: auth.AccountInfo{
			FutuID:      "futu-1",
			MemberID:    "member-1",
			AccountNo:   "account-1",
			AccountID:   "account-id-1",
			AccountType: "margin",
		},
		QuotePermissions: []string{"HK", "US"},
	}
	status := newAuthStatus(store, "/tmp/token.enc")
	if status.Token.Status != statusLoggedIn {
		t.Fatalf("status = %q", status.Token.Status)
	}
	if status.Token.LoggedInAt == nil || status.Token.ExpiresAt == nil {
		t.Fatal("expected login and expiry timestamps")
	}
	if status.Account == nil {
		t.Fatalf("account = %#v", status.Account)
	}
	if status.Account.AccountID != "account-id-1" || status.Account.AccountType != "margin" {
		t.Fatalf("non-sensitive account fields were not preserved: %#v", status.Account)
	}
	if status.Account.FutuID != "" || status.Account.MemberID != "" || status.Account.AccountNo != "" {
		t.Fatalf("sensitive user IDs must not be exposed: %#v", status.Account)
	}
	if len(status.QuotePermissions) != 2 {
		t.Fatalf("quote permissions = %#v", status.QuotePermissions)
	}
}

func TestNewAuthStatusOmitsAccountWithOnlyUserIdentifiers(t *testing.T) {
	store := &auth.TokenStore{
		ExpiresAt: time.Now().Add(time.Hour),
		Account: auth.AccountInfo{
			FutuID:    "futu-1",
			MemberID:  "member-1",
			AccountNo: "account-1",
		},
	}
	status := newAuthStatus(store, "/tmp/token.enc")
	if status.Account != nil {
		t.Fatalf("account with only sensitive user IDs must be omitted: %#v", status.Account)
	}
}

func TestNewAuthStatusExpiredToken(t *testing.T) {
	store := &auth.TokenStore{ExpiresAt: time.Now().Add(-time.Minute)}
	status := newAuthStatus(store, "/tmp/token.enc")
	if status.Token.Status != statusExpired {
		t.Fatalf("status = %q", status.Token.Status)
	}
}

func TestNewAuthStatusWithoutToken(t *testing.T) {
	status := newAuthStatus(nil, "/tmp/token.enc")
	if status.Token.Status != statusNotLoggedIn {
		t.Fatalf("status = %q", status.Token.Status)
	}
	if status.Token.LoggedInAt != nil || status.Token.ExpiresAt != nil {
		t.Fatal("timestamps must be nil without a token")
	}
}

func TestRunLogoutDeletesLocalToken(t *testing.T) {
	keyring.MockInit()
	previousTokenFile := tokenFile
	t.Cleanup(func() {
		tokenFile = previousTokenFile
	})

	path := t.TempDir() + "/token.json"
	store := &auth.TokenStore{
		AccessToken: "access-token",
		ClientID:    "stored-client-id",
	}
	if err := store.Save(path); err != nil {
		t.Fatalf("save token: %v", err)
	}

	tokenFile = path
	if err := runLogout(nil, nil); err != nil {
		t.Fatalf("runLogout() error = %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("local token still exists after revoke failure: %v", statErr)
	}
}
