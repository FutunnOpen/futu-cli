package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
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
	Use:     "quote <symbol...>",
	GroupID: quoteGroupID,
	Short:   "Get real-time quotes",
	Long: `Get real-time quotes for one or more symbols. For US stocks, pre-market, post-market, or overnight sessions are printed as additional rows when available.

Symbol format:
  MARKET.CODE, for example US.AAPL or HK.00700`,
	Example: `  futu quote HK.00700
  futu quote US.AAPL
  futu quote US.AAPL --format json`,
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

func parseOptionalFutuCode(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	code, err := parseOneFutuCode(raw)
	if err != nil {
		return "", err
	}
	return code, nil
}

func renderQuoteTable(quotes []service.StockQuote) {
	tw := output.NewTableWriter(os.Stdout,
		quoteHeaderCode,
		quoteHeaderName,
		"Session",
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
			"REGULAR",
			fmt.Sprintf("%.3f", q.LastPrice),
			formatChange(change),
			formatChangePct(changePct),
			fmt.Sprintf("%.3f", q.HighPrice),
			fmt.Sprintf("%.3f", q.LowPrice),
			fmt.Sprintf("%d", q.Volume),
			formatTurnover(q.Turnover),
		)
		addQuoteSessionRow(tw, q, "PREMARKET", q.PreMarket)
		addQuoteSessionRow(tw, q, "AFTERHOURS", q.AfterMarket)
		addQuoteSessionRow(tw, q, "OVERNIGHT", q.Overnight)
	}
	tw.Render()
}

func addQuoteSessionRow(tw *output.TableWriter, q service.StockQuote, session string, data *service.MarketSession) {
	if data == nil || !hasQuoteSessionData(data) {
		return
	}
	tw.AddRow(
		q.Code,
		q.SCName,
		session,
		fmt.Sprintf("%.3f", data.Price),
		formatChange(data.ChangeVal),
		formatChangePct(data.ChangeRate),
		fmt.Sprintf("%.3f", data.HighPrice),
		fmt.Sprintf("%.3f", data.LowPrice),
		fmt.Sprintf("%d", data.Volume),
		formatTurnover(data.Turnover),
	)
}

func hasQuoteSessionData(data *service.MarketSession) bool {
	return data.Price != 0 || data.Volume != 0 || data.Turnover != 0 ||
		data.HighPrice != 0 || data.LowPrice != 0
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
		million  = 1e6
		thousand = 1e3
	)
	switch {
	case v >= million:
		return fmt.Sprintf("%.2fM", v/million)
	case v >= thousand:
		return fmt.Sprintf("%.2fK", v/thousand)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
