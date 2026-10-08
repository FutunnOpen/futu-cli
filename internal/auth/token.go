package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// tokenExpiryBuffer is subtracted from the actual expiry to trigger
// proactive refresh before the token truly expires.
const tokenExpiryBuffer = 30 * time.Second

var deleteLegacyToken = DeleteFromKeyring

const missingRefreshTokenMessage = "missing refresh token"

// TokenStore holds the persisted authentication state.
type TokenStore struct {
	AccessToken      string         `json:"access_token"`
	RefreshToken     string         `json:"refresh_token"`
	ExpiresAt        time.Time      `json:"expires_at"`
	LoggedInAt       time.Time      `json:"logged_in_at,omitempty"`
	ClientName       string         `json:"client_name"`
	ClientID         string         `json:"client_id"`
	Scope            string         `json:"scope,omitempty"`
	Account          AccountInfo    `json:"account,omitempty"`
	Permissions      PermissionInfo `json:"permissions"`
	QuotePermissions []string       `json:"quote_permissions,omitempty"`
}

// AccountInfo contains account metadata available locally from OAuth2.
type AccountInfo struct {
	FutuID              string   `json:"futu_id,omitempty"`
	MemberID            string   `json:"member_id,omitempty"`
	AccountID           string   `json:"account_id,omitempty"`
	AccountNo           string   `json:"account_no,omitempty"`
	AccountType         string   `json:"account_type,omitempty"`
	AccountChannel      string   `json:"account_channel,omitempty"`
	Name                string   `json:"name,omitempty"`
	ActivatedPackages   []string `json:"activated_packages,omitempty"`
	UnactivatedPackages []string `json:"unactivated_packages,omitempty"`
}

// PermissionInfo records locally known OAuth and account permissions.
type PermissionInfo struct {
	Quote        bool `json:"quote"`
	Trade        bool `json:"trade"`
	TradeChecked bool `json:"trade_checked,omitempty"`
}

// Save persists the token to an AES-GCM encrypted file.
func (t *TokenStore) Save(path string) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := EncryptAndSave(path, data); err != nil {
		return fmt.Errorf("save token file: %w", err)
	}
	return nil
}

// LoadToken reads and decrypts a token file.
func LoadToken(path string) (*TokenStore, error) {
	data, err := DecryptAndLoad(path)
	if err == nil {
		return unmarshalToken(data)
	}
	return migrateTokenFromKeyring(path, err)
}

func migrateTokenFromKeyring(path string, fileErr error) (*TokenStore, error) {
	data, err := LoadFromKeyring()
	if err != nil {
		return nil, fmt.Errorf("load token file: %w", fileErr)
	}
	store, err := unmarshalToken(data)
	if err != nil {
		return nil, err
	}
	if err := store.Save(path); err != nil {
		return nil, fmt.Errorf("migrate token from keyring: %w", err)
	}
	return store, nil
}

// unmarshalToken deserializes a TokenStore from JSON bytes.
func unmarshalToken(data []byte) (*TokenStore, error) {
	var store TokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &store, nil
}

// DeleteToken removes the token file.
func DeleteToken(path string) error {
	if err := removeFileIfExists(path); err != nil {
		return fmt.Errorf("delete token file: %w", err)
	}
	_ = deleteLegacyToken()
	return nil
}

// IsExpired checks whether the access token needs to be refreshed.
// Returns true if the token has expired or is within the expiry buffer.
func (t *TokenStore) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-tokenExpiryBuffer))
}

// EnsureValid loads the stored token, refreshes it if expired, and returns
// a valid access token. This is intended for use in cobra PreRun hooks.
func EnsureValid(clientID, path string) (string, error) {
	store, err := EnsureValidStore(clientID, path)
	if err != nil {
		return "", err
	}
	return store.AccessToken, nil
}

// EnsureValidStore loads the stored token, refreshes it if expired, persists
// the refreshed token, and returns the usable token store.
func EnsureValidStore(clientID, path string) (*TokenStore, error) {
	store, err := LoadToken(path)
	if err != nil {
		return nil, fmt.Errorf("no stored token: %w", err)
	}
	if !store.IsExpired() {
		return store, nil
	}
	return RefreshStoredToken(resolveClientID(clientID, store), path, store)
}

// RefreshStoredToken refreshes a loaded token store and persists the new token.
func RefreshStoredToken(clientID, path string, store *TokenStore) (*TokenStore, error) {
	if len(store.RefreshToken) == 0 {
		return nil, errors.New(missingRefreshTokenMessage)
	}
	resp, err := RefreshAccessToken(clientID, store.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	if len(resp.RefreshToken) == 0 {
		resp.RefreshToken = store.RefreshToken
	}
	updated := buildUpdatedStore(resp, store, clientID)
	if err := updated.Save(path); err != nil {
		return nil, fmt.Errorf("save refreshed token: %w", err)
	}
	return updated, nil
}

func resolveClientID(clientID string, store *TokenStore) string {
	if len(clientID) > 0 {
		return clientID
	}
	return store.ClientID
}

// buildUpdatedStore creates a new TokenStore from a refresh response.
func buildUpdatedStore(resp *TokenResponse, previous *TokenStore, clientID string) *TokenStore {
	updated := NewTokenStore(resp, previous.ClientName, clientID, previous.LoggedInAt)
	if len(updated.Scope) == 0 {
		updated.Scope = previous.Scope
	}
	if accountInfoEmpty(updated.Account) {
		updated.Account = previous.Account
	}
	if len(updated.QuotePermissions) == 0 {
		updated.QuotePermissions = previous.QuotePermissions
	}
	if previous.Permissions.TradeChecked {
		updated.Permissions.Trade = previous.Permissions.Trade
		updated.Permissions.TradeChecked = true
	}
	return updated
}

// removeFileIfExists removes a file, returning nil if it does not exist.
func removeFileIfExists(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
