package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

// Table header labels for the position command.
const (
	posHeaderCode        = "Code"
	posHeaderName        = "Name"
	posHeaderSide        = "Side"
	posHeaderQty         = "Qty"
	posHeaderCanSellQty  = "Can Sell"
	posHeaderCostPrice   = "Cost Price"
	posHeaderMarketPrice = "Mkt Price"
	posHeaderMarketVal   = "Mkt Value"
	posHeaderPLVal       = "P/L"
	posHeaderPLRatio     = "P/L%"
	posHeaderTodayPL     = "Today P/L"
	posHeaderCurrency    = "Currency"
)

var positionSideNames = map[string]string{
	service.PositionSideLong:  "Long",
	service.PositionSideShort: "Short",
	service.PositionSideNone:  "-",
}

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "List positions",
	Long:  "List positions for the specified account, including quantity, cost price, market value, and P/L.",
	RunE:  runPosition,
}

func init() {
	positionCmd.Flags().StringP("account", "a", "", "Trading account ID")
	positionCmd.Flags().String("symbol", "", "Symbol filter in MARKET.CODE format, for example US.AAPL")
	positionCmd.Flags().String("pl-ratio-min", "", "Minimum P/L ratio, for example -0.1 means >= -10%")
	positionCmd.Flags().String("pl-ratio-max", "", "Maximum P/L ratio, for example 0.2 means <= +20%")
	rootCmd.AddCommand(positionCmd)
}

// runPosition is the top-level handler for the position command.
func runPosition(cmd *cobra.Command, _ []string) error {
	account, err := cmd.Flags().GetString("account")
	if err != nil {
		return fmt.Errorf("reading account flag: %w", err)
	}
	symbol, err := cmd.Flags().GetString("symbol")
	if err != nil {
		return fmt.Errorf("reading symbol flag: %w", err)
	}
	symbol, err = parseOptionalFutuCode(symbol)
	if err != nil {
		return err
	}
	plRatioMin, err := cmd.Flags().GetString("pl-ratio-min")
	if err != nil {
		return fmt.Errorf("reading pl-ratio-min flag: %w", err)
	}
	plRatioMax, err := cmd.Flags().GetString("pl-ratio-max")
	if err != nil {
		return fmt.Errorf("reading pl-ratio-max flag: %w", err)
	}

	account = resolveAccount(account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}

	ctx := cmdContext(cmd)
	filter := service.PositionFilter{
		Code:       symbol,
		PLRatioMin: plRatioMin,
		PLRatioMax: plRatioMax,
	}
	positions, err := tradeSvc.GetPositions(ctx, account, filter)
	if err != nil {
		return fmt.Errorf("fetching positions: %w", err)
	}

	printResult(positions, func() {
		renderPositionTable(positions)
	})
	return nil
}

// renderPositionTable writes position data as a formatted table to stdout.
func renderPositionTable(positions []service.Position) {
	tw := output.NewTableWriter(os.Stdout,
		posHeaderCode,
		posHeaderName,
		posHeaderSide,
		posHeaderQty,
		posHeaderCanSellQty,
		posHeaderCostPrice,
		posHeaderMarketPrice,
		posHeaderMarketVal,
		posHeaderPLVal,
		posHeaderPLRatio,
		posHeaderTodayPL,
		posHeaderCurrency,
	)

	for _, p := range positions {
		tw.AddRow(
			p.Code,
			p.StockName,
			formatPositionSide(p.PositionSide),
			string(p.Qty),
			string(p.CanSellQty),
			formatDecimalOrNA(p.CostPrice, p.CostPriceValid),
			string(p.NominalPrice),
			string(p.MarketVal),
			formatDecimalOrNA(p.PLVal, p.PLValValid),
			formatPLRatio(p.PLRatio, p.PLRatioValid),
			string(p.TodayPLVal),
			p.Currency,
		)
	}
	tw.Render()
}

func formatPositionSide(side string) string {
	if name, ok := positionSideNames[side]; ok {
		return name
	}
	return side
}

func formatDecimalOrNA(val service.Decimal, valid bool) string {
	if !valid {
		return "-"
	}
	return string(val)
}

func formatPLRatio(val service.Decimal, valid bool) string {
	if !valid {
		return "-"
	}
	return string(val) + "%"
}
