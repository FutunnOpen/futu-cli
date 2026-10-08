package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
)

// Balance table header labels.
const (
	balanceHeaderToken    = "Token"
	balanceHeaderBalance  = "Balance"
	balanceHeaderValueUSD = "ValueUSD"
)

// Numeric format strings for balance and value display.
const (
	balanceFormat  = "%.8f"
	valueUSDFormat = "%.2f"
)

// Swap argument position indices.
const (
	swapArgFrom   = 0
	swapArgTo     = 1
	swapArgAmount = 2
)

// Transfer argument position indices.
const (
	transferArgToAddr = 0
	transferArgToken  = 1
	transferArgAmount = 2
)

// Required argument counts for commands.
const (
	swapRequiredArgs     = 3
	transferRequiredArgs = 3
)

// Swap quote table header labels.
const (
	swapHeaderFromToken   = "FromToken"
	swapHeaderToToken     = "ToToken"
	swapHeaderFromAmount  = "FromAmount"
	swapHeaderToAmount    = "ToAmount"
	swapHeaderRate        = "Rate"
	swapHeaderPriceImpact = "PriceImpact"
)

// Swap result table header labels.
const (
	swapResultHeaderTxHash = "TxHash"
	swapResultHeaderStatus = "Status"
)

// Transfer result table header labels.
const (
	transferResultHeaderTxHash = "TxHash"
	transferResultHeaderStatus = "Status"
)

// web3Flags holds flag values shared across web3 subcommands.
var web3Flags struct {
	address string
	execute bool
}

var web3Cmd = &cobra.Command{
	Use:   "web3",
	Short: "Web3 钱包与交易",
	Long:  "Web3 功能：查询余额、兑换代币、转账。",
}

var web3BalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "查询钱包余额",
	Long:  "查询指定地址的代币余额。",
	Args:  cobra.NoArgs,
	RunE:  runWeb3Balance,
}

var web3SwapCmd = &cobra.Command{
	Use:   "swap <from> <to> <amount>",
	Short: "代币兑换",
	Long:  "查看兑换报价或执行代币兑换。使用 --execute 执行实际兑换。",
	Args:  cobra.ExactArgs(swapRequiredArgs),
	RunE:  runWeb3Swap,
}

var web3TransferCmd = &cobra.Command{
	Use:   "transfer <to-address> <token> <amount>",
	Short: "代币转账",
	Long:  "向指定地址转账代币。",
	Args:  cobra.ExactArgs(transferRequiredArgs),
	RunE:  runWeb3Transfer,
}

func init() {
	web3BalanceCmd.Flags().StringVar(&web3Flags.address, "address", "", "钱包地址（默认: 空，由后端选择当前钱包）")
	web3SwapCmd.Flags().BoolVar(&web3Flags.execute, "execute", false, "执行兑换（默认: false，仅查看报价）")

	web3Cmd.AddCommand(web3BalanceCmd, web3SwapCmd, web3TransferCmd)
	rootCmd.AddCommand(web3Cmd)
}

// ---------------------------------------------------------------------------
// balance subcommand
// ---------------------------------------------------------------------------

// runWeb3Balance fetches and displays token balances for the given address.
func runWeb3Balance(cmd *cobra.Command, _ []string) error {
	ctx := cmdContext(cmd)
	balances, err := web3Svc.GetBalance(ctx, web3Flags.address)
	if err != nil {
		return fmt.Errorf("fetching balance: %w", err)
	}

	printResult(balances, func() {
		renderBalanceTable(balances)
	})
	return nil
}

// renderBalanceTable writes token balances as a formatted table to stdout.
func renderBalanceTable(balances []service.TokenBalance) {
	tw := output.NewTableWriter(os.Stdout,
		balanceHeaderToken, balanceHeaderBalance, balanceHeaderValueUSD,
	)
	for i := range balances {
		tw.AddRow(
			balances[i].Token,
			fmt.Sprintf(balanceFormat, balances[i].Balance),
			fmt.Sprintf(valueUSDFormat, balances[i].ValueUSD),
		)
	}
	tw.Render()
}

// ---------------------------------------------------------------------------
// swap subcommand
// ---------------------------------------------------------------------------

// runWeb3Swap handles the swap subcommand: shows a quote or executes a swap.
func runWeb3Swap(cmd *cobra.Command, args []string) error {
	from := args[swapArgFrom]
	to := args[swapArgTo]
	amount, err := parseWeb3Amount(args[swapArgAmount])
	if err != nil {
		return err
	}

	quote, err := fetchAndDisplayQuote(cmd, from, to, amount)
	if err != nil {
		return err
	}

	if !web3Flags.execute {
		return nil
	}
	_ = quote // quote already displayed
	return executeSwapWithConfirmation(cmd, from, to, amount)
}

// fetchAndDisplayQuote retrieves a swap quote and renders it to stderr and stdout.
func fetchAndDisplayQuote(cmd *cobra.Command, from, to, amount string) (*service.SwapQuote, error) {
	ctx := cmdContext(cmd)
	quote, err := web3Svc.GetSwapQuote(ctx, from, to, amount)
	if err != nil {
		return nil, fmt.Errorf("fetching swap quote: %w", err)
	}

	printSwapQuoteSummary(quote)
	printResult(quote, func() {
		renderSwapQuoteTable(quote)
	})
	return quote, nil
}

// printSwapQuoteSummary writes a human-readable swap quote to stderr.
func printSwapQuoteSummary(q *service.SwapQuote) {
	fmt.Fprintf(os.Stderr, "\nSwap Quote:\n")
	fmt.Fprintf(os.Stderr, "  From: "+balanceFormat+" %s\n", q.FromAmt, q.FromToken)
	fmt.Fprintf(os.Stderr, "  To: "+balanceFormat+" %s\n", q.ToAmt, q.ToToken)
	fmt.Fprintf(os.Stderr, "  Rate: "+balanceFormat+"\n", q.Rate)
	fmt.Fprintf(os.Stderr, "  Price Impact: "+valueUSDFormat+"%%\n\n", q.PriceImpact)
}

// renderSwapQuoteTable writes a swap quote as a formatted table to stdout.
func renderSwapQuoteTable(q *service.SwapQuote) {
	tw := output.NewTableWriter(os.Stdout,
		swapHeaderFromToken, swapHeaderToToken,
		swapHeaderFromAmount, swapHeaderToAmount,
		swapHeaderRate, swapHeaderPriceImpact,
	)
	tw.AddRow(
		q.FromToken,
		q.ToToken,
		fmt.Sprintf(balanceFormat, q.FromAmt),
		fmt.Sprintf(balanceFormat, q.ToAmt),
		fmt.Sprintf(balanceFormat, q.Rate),
		fmt.Sprintf(valueUSDFormat, q.PriceImpact),
	)
	tw.Render()
}

// executeSwapWithConfirmation prompts for confirmation, then executes the swap.
func executeSwapWithConfirmation(cmd *cobra.Command, from, to, amount string) error {
	if !confirmWeb3Action("Execute swap?") {
		fmt.Fprintln(os.Stderr, "Swap cancelled.")
		return nil
	}

	ctx := cmdContext(cmd)
	result, err := web3Svc.ExecuteSwap(ctx, from, to, amount)
	if err != nil {
		return fmt.Errorf("executing swap: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Swap executed! TxHash: %s\n", result.TxHash)
	printResult(result, func() {
		renderSwapResultTable(result)
	})
	return nil
}

// renderSwapResultTable writes a swap result as a formatted table to stdout.
func renderSwapResultTable(r *service.SwapResult) {
	tw := output.NewTableWriter(os.Stdout,
		swapResultHeaderTxHash, swapResultHeaderStatus,
		swapHeaderFromToken, swapHeaderToToken,
		swapHeaderFromAmount, swapHeaderToAmount,
	)
	tw.AddRow(
		r.TxHash,
		r.Status,
		r.FromToken,
		r.ToToken,
		fmt.Sprintf(balanceFormat, r.FromAmt),
		fmt.Sprintf(balanceFormat, r.ToAmt),
	)
	tw.Render()
}

// ---------------------------------------------------------------------------
// transfer subcommand
// ---------------------------------------------------------------------------

// runWeb3Transfer handles the transfer subcommand with confirmation.
func runWeb3Transfer(cmd *cobra.Command, args []string) error {
	toAddr := args[transferArgToAddr]
	token := args[transferArgToken]
	amount, err := parseWeb3Amount(args[transferArgAmount])
	if err != nil {
		return err
	}

	printTransferSummary(toAddr, token, amount)

	if !confirmWeb3Action("Confirm transfer?") {
		fmt.Fprintln(os.Stderr, "Transfer cancelled.")
		return nil
	}

	return submitTransfer(cmd, toAddr, token, amount)
}

// printTransferSummary writes a human-readable transfer summary to stderr.
func printTransferSummary(toAddr, token, amount string) {
	fmt.Fprintf(os.Stderr, "\nTransfer Summary:\n")
	fmt.Fprintf(os.Stderr, "  To:     %s\n", toAddr)
	fmt.Fprintf(os.Stderr, "  Token:  %s\n", token)
	fmt.Fprintf(os.Stderr, "  Amount: %s\n\n", amount)
}

// submitTransfer calls the web3 service and displays the result.
func submitTransfer(cmd *cobra.Command, toAddr, token, amount string) error {
	ctx := cmdContext(cmd)
	result, err := web3Svc.Transfer(ctx, toAddr, token, amount)
	if err != nil {
		return fmt.Errorf("submitting transfer: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Transfer submitted! TxHash: %s\n", result.TxHash)
	printResult(result, func() {
		renderTransferResultTable(result)
	})
	return nil
}

// renderTransferResultTable writes a transfer result as a formatted table.
func renderTransferResultTable(r *service.TransferResult) {
	tw := output.NewTableWriter(os.Stdout,
		transferResultHeaderTxHash, transferResultHeaderStatus,
	)
	tw.AddRow(r.TxHash, r.Status)
	tw.Render()
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

// confirmWeb3Action prompts the user for y/n confirmation and returns true
// if the user accepts.
func confirmWeb3Action(prompt string) bool {
	return confirmWeb3ActionWithReader(prompt, os.Stdin)
}

func confirmWeb3ActionWithReader(prompt string, input io.Reader) bool {
	fmt.Fprintf(os.Stderr, "%s (y/n): ", prompt)

	reader := bufio.NewReader(input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer := strings.TrimSpace(response)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

func parseWeb3Amount(value string) (string, error) {
	amount := strings.TrimSpace(value)
	if !isPositiveDecimal(amount) {
		return "", fmt.Errorf("amount must be a positive decimal string")
	}
	return amount, nil
}

func isPositiveDecimal(value string) bool {
	if len(value) == 0 {
		return false
	}
	dotSeen := false
	digitAfterDot := false
	positive := false
	for index := range value {
		character := value[index]
		if character == '.' {
			if dotSeen || index == 0 || index == len(value)-1 {
				return false
			}
			dotSeen = true
			continue
		}
		if character < '0' || character > '9' {
			return false
		}
		if dotSeen {
			digitAfterDot = true
		}
		positive = positive || character != '0'
	}
	return positive && (!dotSeen || digitAfterDot)
}
