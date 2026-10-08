package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"time"
)

// OAuth base URL and endpoint paths.
const (
	oauthBaseURL = "https://webapi.futunn.com"
	registerURL  = "/oauth2/register"
	authorizeURL = "/oauth2/authorize/confirm"
	tokenURL     = "/oauth2/token"
)

// Default scopes requested during authorization.
const defaultScopes = "quote:read quote:write trade:read trade:write accid:*"

// Token endpoint auth method for public clients (CLI).
const tokenAuthMethodNone = "none"

// OAuth grant type identifiers.
const (
	grantTypeAuthCode = "authorization_code"
	grantTypeRefresh  = "refresh_token"
)

const (
	clientNameSuffix = " (Futu CLI)"
	stateBytes       = 16
	oauthHTTPTimeout = 30 * time.Second
)

var oauthHTTPClient = &http.Client{Timeout: oauthHTTPTimeout}

// RegistrationResponse holds the response from the client registration endpoint.
type RegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	RegistrationAccessToken string   `json:"registration_access_token"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	Scope                   string   `json:"scope"`
	PKCERequired            bool     `json:"pkce_required"`
}

// TokenResponse holds the response from the token endpoint.
type TokenResponse struct {
	AccessToken         string      `json:"access_token"`
	RefreshToken        string      `json:"refresh_token"`
	ExpiresIn           int         `json:"expires_in"`
	TokenType           string      `json:"token_type"`
	Scope               string      `json:"scope"`
	Account             AccountInfo `json:"account"`
	QuotePermissions    []string    `json:"quote_permissions"`
	FutuID              string      `json:"futu_id"`
	MemberID            string      `json:"member_id"`
	AccountNo           string      `json:"account_no"`
	AccountType         string      `json:"account_type"`
	AccountChannel      string      `json:"account_channel"`
	Name                string      `json:"name"`
	ActivatedPackages   []string    `json:"activated_packages"`
	UnactivatedPackages []string    `json:"unactivated_packages"`
}

// OAuthError represents a structured OAuth error.
type OAuthError struct {
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	return fmt.Sprintf("oauth error: %s — %s", e.Code, e.Description)
}

// oauthErrorResponse represents an error returned by the OAuth token endpoint.
type oauthErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// registrationRequest holds the fields for dynamic client registration.
type registrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope,omitempty"`
}

// RegisterClient performs dynamic client registration (RFC 7591) with the
// Futu OAuth server, returning a client_id for subsequent authorization requests.
func RegisterClient(clientName string, redirectURIs []string) (*RegistrationResponse, error) {
	payload := registrationRequest{
		ClientName:              clientName,
		RedirectURIs:            redirectURIs,
		TokenEndpointAuthMethod: tokenAuthMethodNone,
		Scope:                   defaultScopes,
	}
	body, err := postJSON(oauthBaseURL+registerURL, payload)
	if err != nil {
		return nil, fmt.Errorf("client registration: %w", err)
	}
	return parseRegistrationResponse(body)
}

// BuildAuthorizationURL constructs the browser URL for user authorization.
func BuildAuthorizationURL(clientID, redirectURI string, pkce *PKCEParams, state string) string {
	params := url.Values{
		"client_id":             {clientID},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {pkce.Method},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {defaultScopes},
		"state":                 {state},
	}
	return oauthBaseURL + authorizeURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens using PKCE.
func ExchangeCode(clientID, code, redirectURI, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {grantTypeAuthCode},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}
	body, err := postForm(oauthBaseURL+tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return parseTokenResponseOrError(body)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(clientID, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {grantTypeRefresh},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}
	body, err := postForm(oauthBaseURL+tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	return parseTokenResponseOrError(body)
}

// GenerateState produces a cryptographically random hex string for CSRF protection.
func GenerateState() (string, error) {
	buf := make([]byte, stateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// BuildClientName constructs the OAuth client_name field.
func BuildClientName(override string) string {
	if len(override) > 0 {
		return override + clientNameSuffix
	}
	return defaultClientIdentity() + clientNameSuffix
}

// defaultClientIdentity returns "username@hostname" for the current machine.
func defaultClientIdentity() string {
	hostname, _ := os.Hostname()
	u, _ := user.Current()
	username := "unknown"
	if u != nil {
		username = u.Username
	}
	return username + "@" + hostname
}

// postJSON sends a POST request with a JSON body and returns the response bytes.
func postJSON(targetURL string, payload any) ([]byte, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}
	resp, err := oauthHTTPClient.Post(targetURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("HTTP POST %s: %w", targetURL, err)
	}
	defer resp.Body.Close()
	return readResponseBody(resp)
}

// postForm sends a POST request with a form-urlencoded body and returns the response bytes.
func postForm(targetURL string, data url.Values) ([]byte, error) {
	resp, err := oauthHTTPClient.PostForm(targetURL, data)
	if err != nil {
		return nil, fmt.Errorf("HTTP POST %s: %w", targetURL, err)
	}
	defer resp.Body.Close()
	return readResponseBody(resp)
}

// readResponseBody reads the full response body and checks for HTTP-level errors.
func readResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// parseRegistrationResponse deserializes a RegistrationResponse from JSON.
func parseRegistrationResponse(body []byte) (*RegistrationResponse, error) {
	var resp RegistrationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse registration response: %w", err)
	}
	return &resp, nil
}

// parseTokenResponse deserializes a TokenResponse from JSON.
func parseTokenResponse(body []byte) (*TokenResponse, error) {
	var resp TokenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	return &resp, nil
}

// parseTokenResponseOrError parses a token response, returning an OAuthError
// if the body contains an error code.
func parseTokenResponseOrError(body []byte) (*TokenResponse, error) {
	var errResp oauthErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && len(errResp.Error) > 0 {
		return nil, &OAuthError{
			Code:        errResp.Error,
			Description: errResp.Description,
		}
	}
	return parseTokenResponse(body)
}
