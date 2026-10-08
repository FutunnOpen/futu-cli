package auth

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestTokenStoreEnrichMetadataFromJWT(t *testing.T) {
	payload := `{"iat":1700000000,"exp":1700003600,"account":{"member_id":"m-1","account_id":"id-1","account_no":"a-1","name":"Ada"},"quote_permissions":["HK","US"]}`
	token := "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
	store := &TokenStore{AccessToken: token, Scope: "quote:read trade:read"}

	store.EnrichMetadata()

	if store.LoggedInAt.Unix() != 1700000000 {
		t.Fatalf("LoggedInAt = %v", store.LoggedInAt)
	}
	if store.ExpiresAt.Unix() != 1700003600 {
		t.Fatalf("ExpiresAt = %v", store.ExpiresAt)
	}
	if store.Account.MemberID != "m-1" || store.Account.AccountNo != "a-1" {
		t.Fatalf("Account = %#v", store.Account)
	}
	if store.Account.AccountID != "id-1" {
		t.Fatalf("AccountID = %q", store.Account.AccountID)
	}
	if !store.Permissions.Quote || !store.Permissions.Trade {
		t.Fatalf("Permissions = %#v", store.Permissions)
	}
	wantPermissions := []string{"HK", "US", "quote:read"}
	if len(store.QuotePermissions) != len(wantPermissions) {
		t.Fatalf("QuotePermissions = %#v", store.QuotePermissions)
	}
	for index, want := range wantPermissions {
		if store.QuotePermissions[index] != want {
			t.Fatalf("QuotePermissions[%d] = %q, want %q", index, store.QuotePermissions[index], want)
		}
	}
}

func TestNewTokenStoreRecordsLoginTime(t *testing.T) {
	before := time.Now()
	store := NewTokenStore(&TokenResponse{ExpiresIn: 3600}, "client", "id", time.Time{})
	if store.LoggedInAt.Before(before) || store.LoggedInAt.After(time.Now()) {
		t.Fatalf("LoggedInAt = %v", store.LoggedInAt)
	}
}

func TestNewTokenStoreUsesTopLevelAccountMetadata(t *testing.T) {
	response := &TokenResponse{
		ExpiresIn: 3600, MemberID: "member-1", AccountNo: "account-1",
		AccountType: "cash", AccountChannel: "futu", Name: "Ada",
		QuotePermissions: []string{"HK"},
	}
	store := NewTokenStore(response, "client", "id", time.Now())
	if store.Account.MemberID != "member-1" || store.Account.AccountNo != "account-1" {
		t.Fatalf("Account = %#v", store.Account)
	}
	if len(store.QuotePermissions) != 3 {
		t.Fatalf("QuotePermissions = %#v", store.QuotePermissions)
	}
}

func TestOpaqueTokenUsesScopePermissions(t *testing.T) {
	store := &TokenStore{AccessToken: "opaque-token", Scope: "quote:read trade:read"}
	store.EnrichMetadata()
	if !store.Permissions.Quote || !store.Permissions.Trade {
		t.Fatalf("Permissions = %#v", store.Permissions)
	}
}
