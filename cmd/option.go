package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Option chain table headers.
const (
	optionHeaderSymbol = "Symbol"
	optionHeaderType   = "Type"
	optionHeaderStrike = "Strike"
	optionHeaderExpiry = "Expiry"
	optionHeaderLast   = "Last"
	optionHeaderIV     = "IV"
	optionHeaderOI     = "OI"
)

var optionCmd = &cobra.Command{
	Use:   "option",
	Short: "期权相关命令",
	Long:  "查看期权链等期权数据。",
}

var chainCmd = &cobra.Command{
	Use:   "chain <symbol>",
	Short: "查看期权链",
	Long:  "查看指定标的的期权链数据。",
	Args:  cobra.ExactArgs(1),
	RunE:  runOptionChain,
}

func init() {
	optionCmd.AddCommand(chainCmd)
	rootCmd.AddCommand(optionCmd)
}

// runOptionChain parses the symbol argument, fetches the option chain, and
// renders the output.
func runOptionChain(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parse symbol: %w", err)
	}

	ctx := cmdContext(cmd)
	items, err := quoteSvc.GetOptionChain(ctx, sym.Full)
	if err != nil {
		return fmt.Errorf("get option chain: %w", err)
	}

	printResult(items, func() {
		renderOptionChainTable(items)
	})
	return nil
}

// renderOptionChainTable writes the option chain items as an aligned table to stdout.
func renderOptionChainTable(items []service.OptionChainItem) {
	tw := output.NewTableWriter(os.Stdout,
		optionHeaderSymbol, optionHeaderType, optionHeaderStrike,
		optionHeaderExpiry, optionHeaderLast, optionHeaderIV, optionHeaderOI,
	)
	for _, item := range items {
		tw.AddRow(
			item.Symbol,
			item.Type,
			fmt.Sprintf("%.2f", item.StrikePrice),
			item.ExpiryDate,
			fmt.Sprintf("%.3f", item.LastPrice),
			fmt.Sprintf("%.2f%%", item.ImpliedVol),
			fmt.Sprintf("%d", item.OpenInt),
		)
	}
	tw.Render()
}
