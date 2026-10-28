package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

// Table header labels for the account list subcommand.
const (
	acctHeaderAccountID = "Account ID"
	acctHeaderType      = "Account Type"
	acctHeaderBroker    = "Broker"
	acctHeaderMarkets   = "Markets"
)

const (
	accountTypeCash   = "cash"
	accountTypeMargin = "margin"
)

const (
	securityFirmFutuSecurities = "FUTUSECURITIES"
	securityFirmFutuSG         = "FUTUSG"
	securityFirmFutuInc        = "FUTUINC"
	securityFirmFutuCA         = "FUTUCA"
	securityFirmFutuMY         = "FUTUMY"
	securityFirmFutuAU         = "FUTUAU"
)

const (
	marketHongKong        = 1
	marketUnitedStates    = 2
	marketChina           = 4
	marketFutures         = 5
	marketSingapore       = 6
	marketCrypto          = 7
	marketAustralia       = 8
	marketForeignExchange = 9
	marketBonds           = 10
	marketMalaysia        = 11
	marketCanada          = 12
	marketFunds           = 14
	marketJapan           = 15
	marketStructuredNote  = 16
	marketEventContracts  = 17
	marketSouthKorea      = 18
)

var securityFirmNames = map[string]string{
	securityFirmFutuSecurities: "Futu Securities (Hong Kong)",
	securityFirmFutuSG:         "moomoo (Singapore)",
	securityFirmFutuInc:        "moomoo (United States)",
	securityFirmFutuCA:         "moomoo (Canada)",
	securityFirmFutuMY:         "moomoo (Malaysia)",
	securityFirmFutuAU:         "moomoo (Australia)",
}

var tradingMarketNames = map[int]string{
	marketHongKong:        "Hong Kong",
	marketUnitedStates:    "United States",
	marketChina:           "China Connect",
	marketFutures:         "Futures",
	marketSingapore:       "Singapore",
	marketCrypto:          "Cryptocurrency",
	marketAustralia:       "Australia",
	marketForeignExchange: "Foreign Exchange",
	marketBonds:           "Bonds",
	marketMalaysia:        "Malaysia",
	marketCanada:          "Canada",
	marketFunds:           "Mutual Funds",
	marketJapan:           "Japan",
	marketStructuredNote:  "Structured Notes",
	marketEventContracts:  "Event Contracts",
	marketSouthKorea:      "South Korea",
}

var accountTypeNames = map[string]string{
	accountTypeCash:   "Cash",
	accountTypeMargin: "Margin",
}

// Table header labels for the account funds subcommand.
const (
	defaultFundsCurrency  = "USD"
	fundsHeaderKey        = "Key"
	fundsHeaderValue      = "Value"
	fundsLabelCurrency    = "Currency"
	fundsLabelCash        = "Cash"
	fundsLabelTotalAssets = "Total Assets"
	fundsLabelMarketValue = "Market Value"
	fundsLabelAvailable   = "Available Funds"
	fundsLabelPower       = "Buying Power"
	fundsLabelWithdrawal  = "Max Withdrawal"
	fundsLabelUnrealized  = "Unrealized P/L"
	fundsLabelRealized    = "Realized P/L"
	fundsLabelRiskStatus  = "Risk Status"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account management",
	Long:  "List trading accounts and query account funds.",
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts",
	Long:  "List all trading accounts available to the current user.",
	RunE:  runAccountList,
}

var accountFundsCmd = &cobra.Command{
	Use:   "funds",
	Short: "Query account funds",
	Long:  "Query a fund overview for the specified account, including cash, total assets, market value, and available funds.",
	RunE:  runAccountFunds,
}

var accountCashLogsCmd = &cobra.Command{
	Use:   "cash-logs",
	Short: "List account cash movement logs",
	Long:  "Query paginated cash logs for an account. Timestamps use microseconds.",
	RunE:  runAccountCashLogs,
}

func init() {
	accountFundsCmd.Flags().StringP("account", "a", "", "Trading account ID")
	accountFundsCmd.Flags().StringP("currency", "c", defaultFundsCurrency, "Display currency")
	accountCashLogsCmd.Flags().StringP("account", "a", "", "Trading account ID")
	accountCashLogsCmd.Flags().Int64("begin-time", 0, "Begin time (microseconds, required)")
	accountCashLogsCmd.Flags().Int64("end-time", 0, "End time (microseconds, required)")
	accountCashLogsCmd.Flags().String("direction", "", "Filter: IN or OUT")
	accountCashLogsCmd.Flags().String("code", "", "Filter by symbol MARKET.CODE")
	accountCashLogsCmd.Flags().String("page-flag", "", "Pagination cursor from previous page")
	accountCashLogsCmd.Flags().Int("page-size", 10, "Page size (max 20)")
	_ = accountCashLogsCmd.MarkFlagRequired("begin-time")
	_ = accountCashLogsCmd.MarkFlagRequired("end-time")
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountFundsCmd)
	accountCmd.AddCommand(accountCashLogsCmd)
	rootCmd.AddCommand(accountCmd)
}

// runAccountList is the top-level handler for the account list subcommand.
func runAccountList(cmd *cobra.Command, _ []string) error {
	ctx := cmdContext(cmd)
	accounts, err := tradeSvc.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("fetching accounts: %w", err)
	}

	printResult(accounts, func() {
		renderAccountTable(accounts)
	})
	return nil
}

// renderAccountTable writes account data as a formatted table to stdout.
func renderAccountTable(accounts []service.Account) {
	tw := output.NewTableWriter(os.Stdout,
		acctHeaderAccountID,
		acctHeaderType,
		acctHeaderBroker,
		acctHeaderMarkets,
	)

	for _, a := range accounts {
		tw.AddRow(
			a.AccountID,
			formatAccountType(a.Type),
			formatSecurityFirm(a.SecurityFirm),
			formatTradingMarkets(a.EnabledTradingMarkets),
		)
	}
	tw.Render()
}

func formatAccountType(value string) string {
	if name, ok := accountTypeNames[value]; ok {
		return name
	}
	return value
}

func formatSecurityFirm(value string) string {
	if name, ok := securityFirmNames[value]; ok {
		return name
	}
	return value
}

func formatTradingMarkets(markets []int) string {
	values := make([]string, 0, len(markets))
	for _, market := range markets {
		if name, ok := tradingMarketNames[market]; ok {
			values = append(values, name)
			continue
		}
		values = append(values, strconv.Itoa(market))
	}
	return strings.Join(values, ", ")
}

func filterAccountsByMarket(accounts []service.Account, targetMarket int) []service.Account {
	filtered := make([]service.Account, 0, len(accounts))
	for _, account := range accounts {
		if hasTradingMarket(account.EnabledTradingMarkets, targetMarket) {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func hasTradingMarket(markets []int, targetMarket int) bool {
	for _, market := range markets {
		if market == targetMarket {
			return true
		}
	}
	return false
}

// runAccountFunds is the top-level handler for the account funds subcommand.
func runAccountFunds(cmd *cobra.Command, _ []string) error {
	account, err := cmd.Flags().GetString("account")
	if err != nil {
		return fmt.Errorf("reading account flag: %w", err)
	}

	currency, err := cmd.Flags().GetString("currency")
	if err != nil {
		return fmt.Errorf("reading currency flag: %w", err)
	}
	account = resolveAccount(account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	ctx := cmdContext(cmd)
	funds, err := tradeSvc.GetFunds(ctx, account, currency)
	if err != nil {
		return fmt.Errorf("fetching funds: %w", err)
	}

	printResult(funds, func() {
		renderFundsTable(funds)
	})
	return nil
}

// renderFundsTable writes funds data as a formatted table to stdout.
func runAccountCashLogs(cmd *cobra.Command, _ []string) error {
	account, err := cmd.Flags().GetString("account")
	if err != nil {
		return fmt.Errorf("reading account flag: %w", err)
	}
	account = resolveAccount(account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	beginTime, err := cmd.Flags().GetInt64("begin-time")
	if err != nil {
		return fmt.Errorf("reading begin-time: %w", err)
	}
	endTime, err := cmd.Flags().GetInt64("end-time")
	if err != nil {
		return fmt.Errorf("reading end-time: %w", err)
	}
	direction, err := cmd.Flags().GetString("direction")
	if err != nil {
		return fmt.Errorf("reading direction: %w", err)
	}
	code, err := cmd.Flags().GetString("code")
	if err != nil {
		return fmt.Errorf("reading code: %w", err)
	}
	pageFlag, err := cmd.Flags().GetString("page-flag")
	if err != nil {
		return fmt.Errorf("reading page-flag: %w", err)
	}
	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return fmt.Errorf("reading page-size: %w", err)
	}
	ctx := cmdContext(cmd)
	page, err := tradeSvc.ListCashLogs(ctx, service.CashLogsRequest{
		Account:   account,
		BeginTime: beginTime,
		EndTime:   endTime,
		Direction: direction,
		Code:      code,
		PageFlag:  pageFlag,
		PageSize:  pageSize,
	})
	if err != nil {
		return fmt.Errorf("fetching cash logs: %w", err)
	}
	printResult(page, func() {
		renderCashLogsTable(page)
	})
	return nil
}

func renderCashLogsTable(page *service.CashLogsPage) {
	tw := output.NewTableWriter(os.Stdout, "Time", "Transaction", "Change", "Balance", "Symbol")
	for _, log := range page.CashLogs {
		tw.AddRow(log.CreatedTime, log.TransactionName, log.CashChange, log.Balance, log.Symbol)
	}
	tw.Render()
	if page.PageFlag != "" {
		fmt.Fprintf(os.Stdout, "page_flag: %s completed: %v\n", page.PageFlag, page.Completed)
	}
}

func renderFundsTable(funds *service.Funds) {
	tw := output.NewTableWriter(os.Stdout, fundsHeaderKey, fundsHeaderValue)
	tw.AddRow(fundsLabelCurrency, funds.Currency)
	tw.AddRow(fundsLabelTotalAssets, string(funds.TotalAssets))
	tw.AddRow(fundsLabelCash, string(funds.Cash))
	tw.AddRow(fundsLabelMarketValue, string(funds.MarketValue))
	tw.AddRow(fundsLabelAvailable, string(funds.AvailableFunds))
	tw.AddRow(fundsLabelPower, string(funds.Power))
	tw.AddRow(fundsLabelWithdrawal, string(funds.MaxWithdrawal))
	tw.AddRow(fundsLabelUnrealized, string(funds.UnrealizedPL))
	tw.AddRow(fundsLabelRealized, string(funds.RealizedPL))
	tw.AddRow(fundsLabelRiskStatus, funds.RiskStatus)
	tw.Render()
}
