package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Table header labels for the quote command.
const (
	quoteHeaderCode      = "Code"
	quoteHeaderName      = "Name"
	quoteHeaderLast      = "Last"
	quoteHeaderChange    = "Change"
	quoteHeaderChangePct = "Change%"
	quoteHeaderVolume    = "Volume"
	quoteHeaderTurnover  = "Turnover"
	quoteHeaderHigh      = "High"
	quoteHeaderLow       = "Low"
)

var quoteCmd = &cobra.Command{
	Use:   "quote <symbol...>",
	Short: "获取实时报价快照",
	Long: `获取一个或多个标的的实时报价，包含最新价、涨跌额、涨跌幅、成交量等。

支持的代码格式:
  HK.09988     富途格式（市场.代码）
  9988         纯数字默认港股
  700.HK       代码.市场
  AAPL         纯字母默认美股
  US.AAPL      富途格式美股`,
	Args: cobra.MinimumNArgs(1),
	RunE: runQuote,
}

func init() {
	rootCmd.AddCommand(quoteCmd)
}

func runQuote(cmd *cobra.Command, args []string) error {
	codes, err := parseFutuCodes(args)
	if err != nil {
		return err
	}

	ctx := cmdContext(cmd)
	quotes, err := quoteSvc.GetStockQuote(ctx, codes)
	if err != nil {
		return fmt.Errorf("fetching quotes: %w", err)
	}

	printResult(quotes, func() {
		renderQuoteTable(quotes)
	})
	return nil
}

// parseFutuCodes parses args into Futu API format codes (e.g. "HK.09988").
func parseFutuCodes(args []string) ([]string, error) {
	parsed, err := symbol.ParseMulti(args)
	if err != nil {
		return nil, fmt.Errorf("parsing symbols: %w", err)
	}

	codes := make([]string, 0, len(parsed))
	for _, s := range parsed {
		codes = append(codes, s.FutuCode())
	}
	return codes, nil
}

func renderQuoteTable(quotes []service.StockQuote) {
	tw := output.NewTableWriter(os.Stdout,
		quoteHeaderCode,
		quoteHeaderName,
		quoteHeaderLast,
		quoteHeaderChange,
		quoteHeaderChangePct,
		quoteHeaderHigh,
		quoteHeaderLow,
		quoteHeaderVolume,
		quoteHeaderTurnover,
	)

	for _, q := range quotes {
		change := q.LastPrice - q.PrevClosePrice
		var changePct float64
		if q.PrevClosePrice != 0 {
			changePct = change / q.PrevClosePrice * 100
		}

		tw.AddRow(
			q.Code,
			q.SCName,
			fmt.Sprintf("%.3f", q.LastPrice),
			formatChange(change),
			formatChangePct(changePct),
			fmt.Sprintf("%.3f", q.HighPrice),
			fmt.Sprintf("%.3f", q.LowPrice),
			fmt.Sprintf("%d", q.Volume),
			formatTurnover(q.Turnover),
		)
	}
	tw.Render()
}

func formatChange(v float64) string {
	if v > 0 {
		return fmt.Sprintf("+%.3f", v)
	}
	return fmt.Sprintf("%.3f", v)
}

func formatChangePct(v float64) string {
	if v > 0 {
		return fmt.Sprintf("+%.2f%%", v)
	}
	return fmt.Sprintf("%.2f%%", v)
}

func formatTurnover(v float64) string {
	const (
		billion = 1e8
		tenK    = 1e4
	)
	switch {
	case v >= billion:
		return fmt.Sprintf("%.2f亿", v/billion)
	case v >= tenK:
		return fmt.Sprintf("%.2f万", v/tenK)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
