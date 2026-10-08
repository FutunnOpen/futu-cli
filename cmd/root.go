package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/analytics"
	"github.com/FutunnOpen/futu-cli/internal/auth"
	"github.com/FutunnOpen/futu-cli/internal/client"
	"github.com/FutunnOpen/futu-cli/internal/config"
	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/reporting"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/internal/update"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Output format constants.
const (
	formatTable = "table"
	formatJSON  = "json"
)

const (
	reportSourceModeCLI     = "cli"
	reportSourceModeSkill   = "skill"
	reportSourceFutuCLI     = "1"
	reportSourceFutuSkill   = "7"
	internalSourceFlag      = "--source"
	internalSourcePrefix    = "--source="
	internalAgentFlag       = "--agent"
	internalAgentPrefix     = "--agent="
	agentNameCLI            = "cli"
	agentNameCodex          = "codex"
	agentNameClaudeCode     = "claude_code"
	agentNameUnknown        = "unknown"
	envCodexSessionID       = "CODEX_SESSION_ID"
	envCodexVersion         = "CODEX_VERSION"
	envCodexSandbox         = "CODEX_SANDBOX"
	envClaudeCode           = "CLAUDECODE"
	envClaudeCodeAlt        = "CLAUDE_CODE"
	envClaudeCodeSession    = "CLAUDE_CODE_SESSION_ID"
	envClaudeCodeEntrypoint = "CLAUDE_CODE_ENTRYPOINT"
)

// Analytics endpoint path.
const analyticsEndpoint = "/v1/analytics/cli"

// Unix timestamp magnitude thresholds.
const (
	millisecondTimestampMin   = int64(1_000_000_000_000)
	microsecondTimestampMin   = int64(1_000_000_000_000_000)
	nanosecondTimestampMin    = int64(1_000_000_000_000_000_000)
	nanosecondsPerMillisecond = int64(time.Millisecond)
	nanosecondsPerMicrosecond = int64(time.Microsecond)
)

// Package-level shared state populated by PersistentPreRunE.
var (
	outputFormat     string
	reportSourceMode string
	agentName        string
	tokenFile        string
	cfg              *config.Config
	httpClient       *client.Client
	quoteSvc         *service.QuoteService
	tradeSvc         *service.TradeService
	cryptoSvc        *service.CryptoService
	web3Svc          *service.Web3Service
	reporter         *analytics.Reporter
	requestReporter  *reporting.Reporter
	cmdStartTime     time.Time
)

var (
	codexAgentEnvKeys = []string{
		envCodexSessionID,
		envCodexVersion,
		envCodexSandbox,
	}
	claudeCodeAgentEnvKeys = []string{
		envClaudeCode,
		envClaudeCodeAlt,
		envClaudeCodeSession,
		envClaudeCodeEntrypoint,
	}
)

// Commands that do not require authentication.
var noAuthCommands = map[string]bool{
	"auth":       true,
	"login":      true,
	"logout":     true,
	"status":     true,
	"version":    true,
	"completion": true,
	"config":     true,
	"check":      true,
	"update":     true,
	"help":       true,
}

var noConfigCommands = map[string]bool{
	"version":    true,
	"completion": true,
	"update":     true,
	"help":       true,
}

var rootCmd = &cobra.Command{
	Use:   "futu",
	Short: "Futu CLI — 富途命令行工具",
	Long:  "Futu CLI V1: REST API 行情、交易、Web3，开箱即用。",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cmdStartTime = time.Now()
		return initSharedState(cmd)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		reportAndCheckUpdate(cmd)
	},
}

// Execute runs the root command.
func Execute() error {
	args, internal, err := extractInternalInvocationArgs(os.Args[1:])
	if err != nil {
		return err
	}
	reportSourceMode = internal.source
	agentName = internal.agent
	rootCmd.SetArgs(args)
	defer flushRequestReporter()
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", formatTable,
		"输出格式: table, json（默认: table）")
	rootCmd.PersistentFlags().StringVar(&tokenFile, "token-file", "",
		"OAuth token 加密文件路径")
	_ = rootCmd.PersistentFlags().MarkHidden("token-file")
}

// initSharedState loads config, ensures auth (when needed), and initializes
// the HTTP client, services, and analytics reporter.
func initSharedState(cmd *cobra.Command) error {
	if commandOrParentMatches(cmd, noConfigCommands) {
		return nil
	}
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reporter = analytics.NewReporter(
		cfg.APIBase+analyticsEndpoint,
		cfg.Telemetry,
	)
	if cfg.AutoUpdate {
		update.RefreshCacheIfStale(Version)
	}

	if requiresAuth(cmd) {
		return initAuthenticatedClient()
	}
	return nil
}

// requiresAuth returns false for commands that can run without a token.
// It walks up the command tree so that subcommands (e.g. "config init")
// inherit the no-auth status of their parent.
func requiresAuth(cmd *cobra.Command) bool {
	return !commandOrParentMatches(cmd, noAuthCommands)
}

func commandOrParentMatches(cmd *cobra.Command, commands map[string]bool) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if commands[current.Name()] {
			return true
		}
	}
	return false
}

// initAuthenticatedClient ensures credentials are configured and sets up the
// HTTP client and all service instances using OAuth2.
func initAuthenticatedClient() error {
	return initOAuthClient()
}

// initOAuthClient sets up the client using OAuth token authentication.
func initOAuthClient() error {
	if len(cfg.ClientID) == 0 {
		return fmt.Errorf("client_id 未配置\n请运行 'futu config init' 完成 OAuth 客户端注册")
	}

	token, err := auth.EnsureValid(cfg.ClientID, effectiveTokenFile())
	if err != nil {
		return fmt.Errorf("authentication required: %w\nRun 'futu auth login' to authenticate", err)
	}
	reportSource, err := reportSourceHeaderValue()
	if err != nil {
		return err
	}
	httpClient = client.New(
		cfg.APIBase,
		client.WithToken(token),
		client.WithClientName(auth.BuildClientName("")),
		client.WithVersion(Version),
		client.WithReportSource(reportSource),
		client.WithReportMetadata("futu", effectiveAgentName()),
		client.WithRequestReporter(newRequestReporter(token)),
	)
	initServices()
	return nil
}

func newRequestReporter(token string) *reporting.Reporter {
	requestReporter = reporting.NewReporter(reporting.Options{
		Enabled: cfg.Telemetry,
		Host:    cfg.ReportHost,
		Token:   token,
	})
	return requestReporter
}

func reportSourceHeaderValue() (string, error) {
	mode := effectiveReportSourceMode()
	switch mode {
	case reportSourceModeCLI:
		return reportSourceFutuCLI, nil
	case reportSourceModeSkill:
		return reportSourceFutuSkill, nil
	default:
		return "", fmt.Errorf("invalid report source %q: supported values are %s and %s",
			mode, reportSourceModeCLI, reportSourceModeSkill)
	}
}

func effectiveReportSourceMode() string {
	if len(reportSourceMode) > 0 {
		return reportSourceMode
	}
	return reportSourceModeCLI
}

func effectiveAgentName() string {
	if len(agentName) > 0 {
		return agentName
	}
	if effectiveReportSourceMode() != reportSourceModeSkill {
		return agentNameCLI
	}
	if detected := detectAgentNameFromEnv(); len(detected) > 0 {
		return detected
	}
	return agentNameUnknown
}

func detectAgentNameFromEnv() string {
	if hasAnyNonEmptyEnv(codexAgentEnvKeys) {
		return agentNameCodex
	}
	if hasAnyNonEmptyEnv(claudeCodeAgentEnvKeys) {
		return agentNameClaudeCode
	}
	return ""
}

func hasAnyNonEmptyEnv(keys []string) bool {
	for _, key := range keys {
		if len(os.Getenv(key)) > 0 {
			return true
		}
	}
	return false
}

type internalInvocationArgs struct {
	source string
	agent  string
}

func extractInternalInvocationArgs(args []string) ([]string, internalInvocationArgs, error) {
	filtered := make([]string, 0, len(args))
	internal := internalInvocationArgs{}
	foundSource := false
	foundAgent := false
	for index := 0; index < len(args); index++ {
		arg, value, consumed, ok, err := parseInternalArg(args, index)
		if err != nil {
			return nil, internalInvocationArgs{}, err
		}
		if !ok {
			filtered = append(filtered, args[index])
			continue
		}
		if consumed {
			index++
		}
		if err := applyInternalArg(arg, value, &internal, &foundSource, &foundAgent); err != nil {
			return nil, internalInvocationArgs{}, err
		}
	}
	return filtered, internal, nil
}

func parseInternalArg(args []string, index int) (string, string, bool, bool, error) {
	arg := args[index]
	switch {
	case arg == internalSourceFlag || arg == internalAgentFlag:
		if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
			return "", "", false, false, fmt.Errorf("%s requires a value", arg)
		}
		return arg, args[index+1], true, true, nil
	case strings.HasPrefix(arg, internalSourcePrefix):
		return parseInternalEqualsArg(internalSourceFlag, internalSourcePrefix, arg)
	case strings.HasPrefix(arg, internalAgentPrefix):
		return parseInternalEqualsArg(internalAgentFlag, internalAgentPrefix, arg)
	default:
		return "", "", false, false, nil
	}
}

func parseInternalEqualsArg(flag, prefix, arg string) (string, string, bool, bool, error) {
	value := strings.TrimPrefix(arg, prefix)
	if len(value) == 0 {
		return "", "", false, false, fmt.Errorf("%s requires a value", flag)
	}
	return flag, value, false, true, nil
}

func applyInternalArg(arg, value string, internal *internalInvocationArgs, foundSource, foundAgent *bool) error {
	switch arg {
	case internalSourceFlag:
		if *foundSource {
			return fmt.Errorf("%s may only be specified once", internalSourceFlag)
		}
		if value != reportSourceModeCLI && value != reportSourceModeSkill {
			return fmt.Errorf("invalid report source %q: supported values are %s and %s",
				value, reportSourceModeCLI, reportSourceModeSkill)
		}
		internal.source = value
		*foundSource = true
	case internalAgentFlag:
		normalized, err := normalizeAgentName(value)
		if err != nil {
			return err
		}
		if *foundAgent {
			return fmt.Errorf("%s may only be specified once", internalAgentFlag)
		}
		internal.agent = normalized
		*foundAgent = true
	}
	return nil
}

func normalizeAgentName(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
	switch normalized {
	case agentNameCLI, agentNameCodex, agentNameClaudeCode, agentNameUnknown:
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid agent %q: supported values are %s, %s, %s and %s",
			value, agentNameCLI, agentNameCodex, agentNameClaudeCode, agentNameUnknown)
	}
}

func effectiveTokenFile() string {
	if len(tokenFile) > 0 {
		return tokenFile
	}
	return cfg.TokenFile
}

// initServices creates service instances from the shared HTTP client.
func initServices() {
	quoteSvc = service.NewQuoteService(httpClient)
	tradeSvc = service.NewTradeService(httpClient)
	cryptoSvc = service.NewCryptoService(httpClient)
	web3Svc = service.NewWeb3Service(httpClient)
}

// reportAndCheckUpdate sends analytics and prints an update notice if available.
func reportAndCheckUpdate(cmd *cobra.Command) {
	defer flushRequestReporter()

	if reporter != nil {
		reporter.Report(analytics.Event{
			Command:    cmd.Name(),
			Outcome:    "success",
			DurationMs: time.Since(cmdStartTime).Milliseconds(),
			Version:    Version,
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			AgentName:  effectiveAgentName(),
		})
	}

	if cfg != nil && cfg.AutoUpdate && !isJSONOutput() {
		update.NotifyIfAvailable(Version)
		update.NotifyReleaseNotesIfVersionChanged(Version)
	}
}

func flushRequestReporter() {
	if requestReporter == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), reporting.DefaultFlushTimeout)
	defer cancel()
	_ = requestReporter.Close(ctx)
	requestReporter = nil
}

// isJSONOutput returns true when the user requested JSON output format.
func isJSONOutput() bool {
	return outputFormat == formatJSON
}

// printResult outputs data either as JSON or via the provided table renderer.
func printResult(data any, renderTable func()) {
	if isJSONOutput() {
		_ = output.PrintJSON(os.Stdout, data)
		return
	}
	renderTable()
}

// resolveAccount returns the specified account or falls back to the default.
func resolveAccount(account string) string {
	if len(account) > 0 {
		return account
	}
	if cfg != nil {
		return cfg.DefaultAcct
	}
	return ""
}

// formatTimestamp converts a Unix timestamp to a readable string.
// It accepts seconds, milliseconds, microseconds, or nanoseconds.
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "-"
	}
	switch {
	case ts >= nanosecondTimestampMin:
		return time.Unix(0, ts).Format("2006-01-02 15:04:05")
	case ts >= microsecondTimestampMin:
		return time.Unix(0, ts*nanosecondsPerMicrosecond).Format("2006-01-02 15:04:05")
	case ts >= millisecondTimestampMin:
		return time.Unix(0, ts*nanosecondsPerMillisecond).Format("2006-01-02 15:04:05")
	default:
		return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
	}
}

// cmdContext returns a context annotated with the command name.
func cmdContext(cmd *cobra.Command) context.Context {
	return context.WithValue(
		context.Background(),
		client.CtxKeyCommand,
		cmd.CommandPath(),
	)
}
