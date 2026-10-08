package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/auth"
	"github.com/FutunnOpen/futu-cli/internal/client"
	"github.com/FutunnOpen/futu-cli/internal/config"
	apierr "github.com/FutunnOpen/futu-cli/internal/errors"
	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
)

const (
	statusLoggedIn    = "logged in"
	statusExpired     = "expired"
	statusNotLoggedIn = "not logged in"
	statusNA          = "N/A"
	statusHeaderKey   = "Key"
	statusHeaderValue = "Value"
	labelStatus       = "Status"
	labelExpires      = "Expires"
	labelClientName   = "Client Name"
	labelLoggedInAt   = "Logged In At"
	labelQuoteAccess  = "Quote Access"
	labelTradeAccess  = "Trade Access"
	labelAccountID    = "Account ID"
	labelAccountType  = "Account Type"
	labelAccountChan  = "Account Channel"
	labelAccountName  = "Account Name"
	labelActivePkgs   = "Activated Packages"
	labelInactivePkgs = "Unactivated Packages"
	labelQuotePerms   = "Quote Permissions"
	messageUsingToken = "已登录，继续使用现有 Token。"
	messageRefreshed  = "Token 已刷新。"
	messageLoginOK    = "登录成功。"
)

const authorizationTimeout = 5 * time.Minute

var (
	clientNameFlag string
	noBrowserFlag  bool
)

var authCmd = &cobra.Command{Use: "auth", Short: "管理 OAuth2 认证"}
var loginCmd = &cobra.Command{Use: "login", Short: "OAuth2 授权登录", RunE: runLogin}
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "清除已存储的 OAuth Token",
	Long:  "清除已存储的 OAuth Token。下次执行需要认证的命令时将重新触发登录。",
	Args:  cobra.NoArgs,
	RunE:  runLogout,
}
var statusCmd = &cobra.Command{
	Use: "status", Short: "本地查看 token、账户和行情权限状态", RunE: runStatus,
}
var registerCmd = &cobra.Command{Use: "register", Short: "重新注册 OAuth2 客户端", RunE: runRegister}

func init() {
	loginCmd.Flags().StringVar(&clientNameFlag, "client-name", "", "客户端名称，用于标识 AI agent（默认: 用户名@主机名）")
	loginCmd.Flags().BoolVar(&noBrowserFlag, "no-browser", false, "不自动打开浏览器（默认: false）")
	authCmd.AddCommand(loginCmd, logoutCmd, statusCmd, registerCmd)
	rootCmd.AddCommand(authCmd)
}

func runLogin(_ *cobra.Command, _ []string) error {
	if cfg == nil {
		return fmt.Errorf("configuration not loaded; please check your config file")
	}
	if useExistingToken(effectiveTokenFile()) {
		return nil
	}
	return startLocalLogin(effectiveTokenFile())
}

func useExistingToken(path string) bool {
	store, err := auth.LoadToken(path)
	if err != nil {
		return false
	}
	if !store.IsExpired() {
		syncStoredAccount(path, store)
		fmt.Fprintln(os.Stderr, messageUsingToken)
		return true
	}
	clientID := cfg.ClientID
	if len(clientID) == 0 {
		clientID = store.ClientID
	}
	if _, err := auth.EnsureValid(clientID, path); err != nil {
		fmt.Fprintf(os.Stderr, "现有 token 刷新失败，将重新授权: %v\n", err)
		return false
	}
	if refreshed, err := auth.LoadToken(path); err == nil {
		syncStoredAccount(path, refreshed)
	}
	fmt.Fprintln(os.Stderr, messageRefreshed)
	return true
}

func startLocalLogin(path string) error {
	server, err := newLoginCallbackServer()
	if err != nil {
		return fmt.Errorf("start callback server: %w", err)
	}
	defer server.Shutdown()
	redirectURI := server.RedirectURI()
	clientID, err := ensureClientID([]string{redirectURI})
	if err != nil {
		return err
	}
	request, err := newAuthorizationRequest(clientID, redirectURI)
	if err != nil {
		return err
	}
	if err := presentAuthorizationURL(request.URL); err != nil {
		return err
	}
	result, err := waitForCallback(server, redirectURI)
	if err != nil {
		return err
	}
	return exchangeAndSave(result, request, path)
}

func newLoginCallbackServer() (*auth.CallbackServer, error) {
	if len(cfg.ClientID) > 0 && len(cfg.RedirectURI) > 0 {
		return auth.NewCallbackServerForRedirectURI(cfg.RedirectURI)
	}
	return auth.NewCallbackServer()
}

type authorizationRequest struct {
	ClientID    string
	RedirectURI string
	State       string
	PKCE        *auth.PKCEParams
	URL         string
}

func newAuthorizationRequest(clientID, redirectURI string) (*authorizationRequest, error) {
	pkce, err := auth.GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE: %w", err)
	}
	state, err := auth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	return &authorizationRequest{
		ClientID: clientID, RedirectURI: redirectURI, State: state, PKCE: pkce,
		URL: auth.BuildAuthorizationURL(clientID, redirectURI, pkce, state),
	}, nil
}

func presentAuthorizationURL(authURL string) error {
	fmt.Fprintf(os.Stderr, "请在浏览器中完成授权:\n%s\n", authURL)
	if noBrowserFlag {
		return nil
	}
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "无法自动打开浏览器，请手动访问上面的地址: %v\n", err)
	}
	return nil
}

func waitForCallback(server *auth.CallbackServer, redirectURI string) (*auth.CallbackResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), authorizationTimeout)
	defer cancel()
	if !noBrowserFlag {
		return waitForHTTPCallback(ctx, server)
	}
	fmt.Fprintln(os.Stderr, "如果浏览器无法连接本机回调端口，请复制地址栏中的完整回调 URL 并粘贴到此处：")
	return waitForHTTPOrManualCallback(ctx, server, redirectURI)
}

type callbackOutcome struct {
	result *auth.CallbackResult
	err    error
}

func waitForHTTPCallback(ctx context.Context, server *auth.CallbackServer) (*auth.CallbackResult, error) {
	result, err := server.WaitForCallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("等待授权: %w", err)
	}
	return result, nil

}

func waitForHTTPOrManualCallback(
	ctx context.Context, server *auth.CallbackServer, redirectURI string,
) (*auth.CallbackResult, error) {
	outcomes := make(chan callbackOutcome, 2)
	go func() {
		result, err := server.WaitForCallback(ctx)
		outcomes <- callbackOutcome{result: result, err: err}
	}()
	go readManualCallback(redirectURI, outcomes)
	select {
	case outcome := <-outcomes:
		if outcome.err != nil {
			return nil, fmt.Errorf("等待授权: %w", outcome.err)
		}
		return outcome.result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("等待授权: %w", ctx.Err())
	}
}

func readManualCallback(redirectURI string, outcomes chan<- callbackOutcome) {
	reader := bufio.NewReader(os.Stdin)
	for {
		value, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		result, err := auth.ParseCallbackURL(strings.TrimSpace(value), redirectURI)
		if err != nil {
			fmt.Fprintf(os.Stderr, "回调 URL 无效，请重新粘贴: %v\n", err)
			continue
		}
		outcomes <- callbackOutcome{result: result}
		return
	}
}

func exchangeAndSave(result *auth.CallbackResult, request *authorizationRequest, path string) error {
	if len(result.Error) > 0 {
		return fmt.Errorf("授权失败: %s", result.Error)
	}
	if result.State != request.State {
		return fmt.Errorf("state 校验失败，可能存在 CSRF 攻击")
	}
	response, err := auth.ExchangeCode(
		request.ClientID, result.Code, request.RedirectURI, request.PKCE.CodeVerifier,
	)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}
	return saveLoginToken(response, request.ClientID, path)
}

func ensureClientID(redirectURIs []string) (string, error) {
	if len(cfg.ClientID) > 0 && cfg.RedirectURI == redirectURIs[0] {
		return cfg.ClientID, nil
	}
	return registerAndSaveClientID(redirectURIs)
}

func registerAndSaveClientID(redirectURIs []string) (string, error) {
	response, err := auth.RegisterClient(auth.BuildClientName(clientNameFlag), redirectURIs)
	if err != nil {
		return "", fmt.Errorf("client registration: %w", err)
	}
	if err := cfg.Set(config.KeyClientID, response.ClientID); err != nil {
		return "", fmt.Errorf("set client_id: %w", err)
	}
	if err := cfg.Set(config.KeyRedirectURI, redirectURIs[0]); err != nil {
		return "", fmt.Errorf("set redirect_uri: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return "", fmt.Errorf("save config: %w", err)
	}
	return response.ClientID, nil
}

func runRegister(_ *cobra.Command, _ []string) error {
	server, err := auth.NewCallbackServer()
	if err != nil {
		return fmt.Errorf("start callback server: %w", err)
	}
	redirectURI := server.RedirectURI()
	server.Shutdown()
	_, err = registerAndSaveClientID([]string{redirectURI})
	return err
}

func saveLoginToken(response *auth.TokenResponse, clientID, path string) error {
	store := auth.NewTokenStore(response, auth.BuildClientName(clientNameFlag), clientID, time.Time{})
	if err := store.Save(path); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	syncStoredAccount(path, store)
	fmt.Fprintln(os.Stderr, messageLoginOK)
	return nil
}

func syncStoredAccount(path string, store *auth.TokenStore) {
	store.EnrichMetadata()
	if store.LoggedInAt.IsZero() {
		store.LoggedInAt = tokenFileModTime(path)
	}
	if accountMetadataComplete(store.Account) {
		_ = store.Save(path)
		return
	}
	httpClient := client.New(
		cfg.APIBase,
		client.WithToken(store.AccessToken),
		client.WithClientName(auth.BuildClientName(clientNameFlag)),
		client.WithVersion(Version),
	)
	accounts, err := service.NewTradeService(httpClient).ListAccounts(context.Background())
	if err != nil {
		markTradePermission(store, err)
		_ = store.Save(path)
		return
	}
	if account := selectStatusAccount(accounts); account != nil {
		mergeStatusAccount(&store.Account, account)
		store.Permissions.Trade = true
		store.Permissions.TradeChecked = true
	}
	_ = store.Save(path)
}

func accountMetadataComplete(account auth.AccountInfo) bool {
	return len(account.AccountID) > 0 && len(account.AccountType) > 0
}

func selectStatusAccount(accounts []service.Account) *service.Account {
	if len(accounts) == 0 {
		return nil
	}
	for index := range accounts {
		if accounts[index].AccountID == cfg.DefaultAcct || accounts[index].AccountNo == cfg.DefaultAcct {
			return &accounts[index]
		}
	}
	return &accounts[0]
}

func mergeStatusAccount(target *auth.AccountInfo, source *service.Account) {
	if len(target.FutuID) == 0 {
		target.FutuID = source.FutuID
	}
	if len(target.AccountNo) == 0 {
		target.AccountNo = source.AccountNo
	}
	if len(target.AccountID) == 0 {
		target.AccountID = source.AccountID
	}
	if len(target.MemberID) == 0 {
		target.MemberID = source.MemberID
	}
	if len(target.AccountType) == 0 {
		target.AccountType = source.Type
	}
	if len(target.AccountChannel) == 0 {
		target.AccountChannel = source.AccountChannel
		if len(target.AccountChannel) == 0 {
			target.AccountChannel = source.Channel
		}
		if len(target.AccountChannel) == 0 {
			target.AccountChannel = source.SecurityFirm
		}
	}
	if len(target.Name) == 0 {
		target.Name = source.Name
	}
}

func markTradePermission(store *auth.TokenStore, err error) {
	var cliErr *apierr.CLIError
	if errors.As(err, &cliErr) && cliErr.Code == apierr.CodePermission {
		store.Permissions.Trade = false
		store.Permissions.TradeChecked = true
	}
}

func runLogout(_ *cobra.Command, _ []string) error {
	path := effectiveTokenFile()
	if err := auth.DeleteToken(path); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Logged out successfully.")
	return nil
}

func runStatus(_ *cobra.Command, _ []string) error {
	path := effectiveTokenFile()
	store, err := auth.LoadToken(path)
	if err != nil {
		return printAuthStatus(newAuthStatus(nil, path))
	}
	store, err = refreshTokenIfExpired(path, store)
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}
	store.EnrichMetadata()
	if store.LoggedInAt.IsZero() {
		store.LoggedInAt = tokenFileModTime(path)
	}
	if len(store.Account.AccountID) == 0 {
		store.Account.AccountID = cfg.DefaultAcct
	}
	return printAuthStatus(newAuthStatus(store, path))
}

func refreshTokenIfExpired(path string, store *auth.TokenStore) (*auth.TokenStore, error) {
	if !store.IsExpired() {
		return store, nil
	}
	return auth.RefreshStoredToken(statusRefreshClientID(store), path, store)
}

func statusRefreshClientID(store *auth.TokenStore) string {
	if cfg != nil && len(cfg.ClientID) > 0 {
		return cfg.ClientID
	}
	return store.ClientID
}

func tokenFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

type tokenStatus struct {
	Status     string `json:"status"`
	LoggedInAt *int64 `json:"logged_in_at"`
	ExpiresAt  *int64 `json:"expires_at"`
	ClientName string `json:"client_name,omitempty"`
}

type authStatusData struct {
	Token            tokenStatus       `json:"token"`
	Permissions      permissionStatus  `json:"permissions"`
	Account          *auth.AccountInfo `json:"account,omitempty"`
	QuotePermissions []string          `json:"quote_permissions,omitempty"`
}

type permissionStatus struct {
	Quote bool `json:"quote"`
	Trade bool `json:"trade"`
}

func newAuthStatus(store *auth.TokenStore, _ string) authStatusData {
	status := authStatusData{
		Token:            tokenStatus{Status: statusNotLoggedIn},
		QuotePermissions: []string{},
	}
	if store == nil {
		return status
	}
	status.Token.Status = statusLoggedIn
	if store.IsExpired() {
		status.Token.Status = statusExpired
	}
	status.Token.LoggedInAt = unixTimePointer(store.LoggedInAt)
	status.Token.ExpiresAt = unixTimePointer(store.ExpiresAt)
	status.Token.ClientName = store.ClientName
	status.Permissions = permissionStatus{
		Quote: store.Permissions.Quote,
		Trade: store.Permissions.Trade,
	}
	if accountInfoAvailable(store.Account) {
		account := sanitizeStatusAccount(store.Account)
		status.Account = &account
	}
	status.QuotePermissions = store.QuotePermissions
	return status
}

func sanitizeStatusAccount(account auth.AccountInfo) auth.AccountInfo {
	account.FutuID = ""
	account.MemberID = ""
	account.AccountNo = ""
	return account
}

func unixTimePointer(value time.Time) *int64 {
	if value.IsZero() {
		return nil
	}
	seconds := value.Unix()
	return &seconds
}

func printAuthStatus(status authStatusData) error {
	printResult(status, func() { renderStatusTable(status) })
	return nil
}

func renderStatusTable(status authStatusData) {
	tw := output.NewTableWriter(os.Stdout, statusHeaderKey, statusHeaderValue)
	tw.AddRow(labelStatus, status.Token.Status)
	tw.AddRow(labelLoggedInAt, formatOptionalTime(status.Token.LoggedInAt))
	tw.AddRow(labelExpires, formatOptionalTime(status.Token.ExpiresAt))
	addStatusRow(tw, labelClientName, status.Token.ClientName)
	tw.AddRow(labelQuoteAccess, formatPermission(status.Permissions.Quote))
	tw.AddRow(labelTradeAccess, formatPermission(status.Permissions.Trade))
	if status.Account != nil {
		addAccountStatusRows(tw, *status.Account)
	}
	addStatusRow(tw, labelQuotePerms, strings.Join(status.QuotePermissions, ", "))
	tw.Render()
}

func addAccountStatusRows(tw *output.TableWriter, account auth.AccountInfo) {
	addStatusRow(tw, labelAccountID, account.AccountID)
	addStatusRow(tw, labelAccountType, account.AccountType)
	addStatusRow(tw, labelAccountChan, account.AccountChannel)
	addStatusRow(tw, labelAccountName, account.Name)
	addStatusRow(tw, labelActivePkgs, strings.Join(account.ActivatedPackages, ", "))
	addStatusRow(tw, labelInactivePkgs, strings.Join(account.UnactivatedPackages, ", "))
}

func addStatusRow(tw *output.TableWriter, label, value string) {
	if len(value) > 0 {
		tw.AddRow(label, value)
	}
}

func formatPermission(allowed bool) string {
	if allowed {
		return "yes"
	}
	return "no"
}

func accountInfoAvailable(account auth.AccountInfo) bool {
	return len(account.AccountID) > 0 || len(account.AccountType) > 0 ||
		len(account.AccountChannel) > 0 || len(account.Name) > 0 ||
		len(account.ActivatedPackages) > 0 || len(account.UnactivatedPackages) > 0
}

func formatOptionalTime(value *int64) string {
	if value == nil {
		return statusNA
	}
	return time.Unix(*value, 0).Format(time.RFC3339)
}

func openBrowser(targetURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", targetURL).Start()
	case "linux":
		return exec.Command("xdg-open", targetURL).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", targetURL).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
