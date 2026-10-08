package auth

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const jwtSegmentCount = 3

// NewTokenStore enriches an OAuth2 response with locally available metadata.
func NewTokenStore(
	response *TokenResponse, clientName, clientID string, loggedInAt time.Time,
) *TokenStore {
	if loggedInAt.IsZero() {
		loggedInAt = time.Now()
	}
	account := response.Account
	mergeResponseAccount(&account, response)
	scope := response.Scope
	if len(scope) == 0 {
		scope = defaultScopes
	}
	store := &TokenStore{
		AccessToken: response.AccessToken, RefreshToken: response.RefreshToken,
		ExpiresAt:  time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
		LoggedInAt: loggedInAt, ClientName: clientName, ClientID: clientID,
		Scope: scope, Account: account,
		QuotePermissions: append([]string(nil), response.QuotePermissions...),
	}
	store.updateScopePermissions()
	store.enrichFromAccessToken()
	return store
}

// EnrichMetadata fills missing metadata from JWT claims without network access.
func (t *TokenStore) EnrichMetadata() {
	t.enrichFromAccessToken()
}

func (t *TokenStore) enrichFromAccessToken() {
	if len(t.Scope) == 0 {
		t.Scope = defaultScopes
	}
	claims, ok := decodeJWTClaims(t.AccessToken)
	if !ok {
		t.QuotePermissions = normalizePermissions(t.QuotePermissions, t.Scope)
		t.updateScopePermissions()
		return
	}
	if t.LoggedInAt.IsZero() {
		t.LoggedInAt = claimTime(claims, "iat")
	}
	if t.ExpiresAt.IsZero() {
		t.ExpiresAt = claimTime(claims, "exp")
	}
	if scope, ok := claims["scope"].(string); ok && len(scope) > 0 {
		t.Scope = scope
	}
	fillAccountFromClaims(&t.Account, claims)
	claimPermissions := firstStringSliceClaim(
		claims, "quote_permissions", "quote_permission", "quote_markets", "market_permissions",
	)
	t.QuotePermissions = normalizePermissions(
		append(t.QuotePermissions, claimPermissions...), t.Scope,
	)
	t.updateScopePermissions()
}

func decodeJWTClaims(token string) (map[string]any, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != jwtSegmentCount {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

func fillAccountFromClaims(account *AccountInfo, claims map[string]any) {
	if nested, ok := claims["account"].(map[string]any); ok {
		fillAccountFields(account, nested)
	}
	if nested, ok := claims["user"].(map[string]any); ok {
		fillAccountFields(account, nested)
	}
	fillAccountFields(account, claims)
	fillString(&account.MemberID, claims, "sub")
}

func fillAccountFields(account *AccountInfo, accountClaims map[string]any) {
	fillString(&account.FutuID, accountClaims, "futu_id", "futuId", "user_id", "uid")
	fillString(&account.MemberID, accountClaims, "member_id", "memberId")
	fillString(&account.AccountID, accountClaims, "account_id", "accountId")
	fillString(&account.AccountNo, accountClaims, "account_no", "accountNo")
	fillString(&account.AccountType, accountClaims, "account_type", "accountType")
	fillString(&account.AccountChannel, accountClaims, "account_channel", "accountChannel")
	fillString(&account.Name, accountClaims, "name", "user_name")
	if len(account.ActivatedPackages) == 0 {
		account.ActivatedPackages = firstStringSliceClaim(accountClaims, "activated_packages")
	}
	if len(account.UnactivatedPackages) == 0 {
		account.UnactivatedPackages = firstStringSliceClaim(accountClaims, "unactivated_packages")
	}
}

func (t *TokenStore) updateScopePermissions() {
	for _, scope := range strings.Fields(t.Scope) {
		if strings.HasPrefix(scope, "quote:") {
			t.Permissions.Quote = true
		}
		if strings.HasPrefix(scope, "trade:") && !t.Permissions.TradeChecked {
			t.Permissions.Trade = true
		}
	}
	if len(t.QuotePermissions) > 0 {
		t.Permissions.Quote = true
	}
}

func mergeResponseAccount(account *AccountInfo, response *TokenResponse) {
	if len(account.FutuID) == 0 {
		account.FutuID = response.FutuID
	}
	if len(account.MemberID) == 0 {
		account.MemberID = response.MemberID
	}
	if len(account.AccountNo) == 0 {
		account.AccountNo = response.AccountNo
	}
	if len(account.AccountType) == 0 {
		account.AccountType = response.AccountType
	}
	if len(account.AccountChannel) == 0 {
		account.AccountChannel = response.AccountChannel
	}
	if len(account.Name) == 0 {
		account.Name = response.Name
	}
	if len(account.ActivatedPackages) == 0 {
		account.ActivatedPackages = append([]string(nil), response.ActivatedPackages...)
	}
	if len(account.UnactivatedPackages) == 0 {
		account.UnactivatedPackages = append([]string(nil), response.UnactivatedPackages...)
	}
}

func fillString(target *string, claims map[string]any, keys ...string) {
	if len(*target) > 0 {
		return
	}
	for _, key := range keys {
		if value, ok := claims[key].(string); ok && len(value) > 0 {
			*target = value
			return
		}
	}
}

func firstStringSliceClaim(claims map[string]any, keys ...string) []string {
	for _, key := range keys {
		if values := stringSlice(claims[key]); len(values) > 0 {
			return values
		}
	}
	return nil
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case string:
		return strings.FieldsFunc(typed, func(r rune) bool { return r == ' ' || r == ',' })
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && len(text) > 0 {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func normalizePermissions(permissions []string, scope string) []string {
	seen := make(map[string]struct{})
	for _, permission := range permissions {
		if len(permission) > 0 {
			seen[permission] = struct{}{}
		}
	}
	for _, permission := range strings.Fields(scope) {
		if strings.HasPrefix(permission, "quote:") {
			seen[permission] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for permission := range seen {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func claimTime(claims map[string]any, key string) time.Time {
	seconds, ok := claims[key].(float64)
	if !ok || seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(seconds), 0)
}

func accountInfoEmpty(account AccountInfo) bool {
	return len(account.FutuID) == 0 && len(account.MemberID) == 0 && len(account.AccountID) == 0 &&
		len(account.AccountNo) == 0 &&
		len(account.AccountType) == 0 && len(account.AccountChannel) == 0 &&
		len(account.Name) == 0 && len(account.ActivatedPackages) == 0 &&
		len(account.UnactivatedPackages) == 0
}
