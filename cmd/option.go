package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
)

const (
	defaultOptionFilterStandard = "ALL"
	defaultOptionChainDays      = 30
)

var indexOptionTypeBySymbol = map[string]int{
	"HK.800000": 1,    // HSI
	"HK.800100": 2,    // HSCEI
	"HK.800700": 5,    // HSTECH
	"US.VIX":    1001, // VIX
	"US.VIXW":   1001,
	"JP.N225":   2001,
	"JP.NI225":  2001,
}

var optionChainFlags struct {
	start           string
	end             string
	indexOptionType int
	filterStandard  string
}

var optionCmd = &cobra.Command{
	Use:     "option",
	GroupID: quoteGroupID,
	Short:   "Option commands",
	Long:    "View option chains. If no date is specified, the command uses expirations within the next 30 days; index symbols include index_option_type.",
}

var chainCmd = &cobra.Command{
	Use:   "chain <symbol>",
	Short: "Show option chain",
	Long:  "Get the option chain for the specified symbol and expiration range. Supports HK / US / JP.",
	Example: `  futu option chain HK.00700
  futu option chain US.AAPL
  futu option chain HK.800000
  futu option chain HK.00700 --start 2026-05-01 --end 2026-05-31`,
	Args: cobra.ExactArgs(1),
	RunE: runOptionChain,
}

func init() {
	chainCmd.Flags().StringVar(&optionChainFlags.start, "start", "",
		"Start expiration date yyyy-MM-dd, inclusive (default: earliest expiration within the next 30 days, or the nearest future expiration)")
	chainCmd.Flags().StringVar(&optionChainFlags.end, "end", "",
		fmt.Sprintf("End expiration date yyyy-MM-dd, inclusive (default: latest expiration within the next %d days, or the nearest future expiration)", defaultOptionChainDays))
	chainCmd.Flags().IntVar(&optionChainFlags.indexOptionType, "index-option-type", 0,
		"Index option type (default: inferred from symbol; omitted for stocks). HK 1=HSI 2=HSCEI 3=Mini-HSI 4=Mini-HSCEI 5=HSTECH; US 1001=VIX; JP 2001=N225")
	chainCmd.Flags().StringVar(&optionChainFlags.filterStandard, "filter-standard", defaultOptionFilterStandard,
		"Standard/non-standard filter (default: ALL). ALL / STANDARD / NON_STANDARD")
	optionCmd.AddCommand(chainCmd)
	rootCmd.AddCommand(optionCmd)
}

func runOptionChain(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parse symbol: %w", err)
	}
	indexType, hasType := resolveIndexOptionType(cmd, sym.FutuCode())
	params := service.OptionChainParams{
		IndexOptionType: indexType,
		HasIndexType:    hasType,
		FilterStandard:  optionChainFlags.filterStandard,
	}
	start, end := optionChainFlags.start, optionChainFlags.end
	if start == "" && end == "" {
		dates, err := quoteSvc.GetOptionExpirationDates(cmdContext(cmd), sym.FutuCode(), params)
		if err != nil {
			return err
		}
		start, end = pickOptionChainRange(time.Now(), dates, defaultOptionChainDays)
	}
	params.Start, params.End = start, end
	result, err := quoteSvc.GetOptionChain(cmdContext(cmd), sym.FutuCode(), params)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderOptionChainTable(jsonObjectList(decodeJSONValue(result.Data), "option_chain"))
	})
	return nil
}

func defaultOptionChainRange(now time.Time) (string, string) {
	return now.Format("2006-01-02"), now.AddDate(0, 0, defaultOptionChainDays).Format("2006-01-02")
}

func pickOptionChainRange(now time.Time, dates []string, windowDays int) (string, string) {
	today := now.Format("2006-01-02")
	limit := now.AddDate(0, 0, windowDays).Format("2006-01-02")
	inWindow := make([]string, 0, len(dates))
	future := make([]string, 0, len(dates))
	for _, date := range dates {
		if date < today {
			continue
		}
		future = append(future, date)
		if date <= limit {
			inWindow = append(inWindow, date)
		}
	}
	sort.Strings(inWindow)
	sort.Strings(future)
	if len(inWindow) > 0 {
		return inWindow[0], inWindow[len(inWindow)-1]
	}
	if len(future) > 0 {
		return future[0], future[0]
	}
	return today, limit
}

func resolveIndexOptionType(cmd *cobra.Command, futuCode string) (int, bool) {
	if cmd.Flags().Changed("index-option-type") {
		return optionChainFlags.indexOptionType, true
	}
	if t, ok := indexOptionTypeBySymbol[strings.ToUpper(futuCode)]; ok {
		return t, true
	}
	return 0, false
}

func renderOptionChainTable(items []map[string]any) {
	tw := output.NewTableWriter(os.Stdout,
		"Symbol", "Type", "Strike", "Expiry", "Name",
	)
	for _, item := range items {
		tw.AddRow(
			jsonString(item, "code"),
			jsonString(item, "option_type"),
			formatPrice(jsonFloat(item, "strike_price")),
			jsonString(item, "strike_time"),
			firstNonEmpty(jsonString(item, "sc_name"), jsonString(item, "name")),
		)
	}
	tw.Render()
}
