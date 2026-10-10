package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
)

const defaultTickerCount = 20

var tickerFlags struct {
	count   int
	periods []string
}

var tickerCmd = &cobra.Command{
	Use:     "ticker <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get tick-by-tick trades",
	Long:    "Get the latest tick-by-tick trades for the specified symbol. --period can be repeated: NORMAL / BEFORE / AFTER / OVERNIGHT.",
	Example: `  futu ticker HK.00700
  futu ticker US.AAPL --num 50 --period OVERNIGHT
  futu ticker HK.00700 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runTicker,
}

func init() {
	tickerCmd.Flags().IntVarP(&tickerFlags.count, "num", "n", defaultTickerCount,
		"Return the latest N ticks (default: 20, range 1-750)")
	tickerCmd.Flags().StringArrayVar(&tickerFlags.periods, "period", nil,
		"Filter by trading period (default: no filter; backend returns the current period). Repeatable: NORMAL / BEFORE / AFTER / OVERNIGHT")
	tickerCmd.Flags().IntVarP(&tickerFlags.count, "count", "c", defaultTickerCount,
		"Same as --num (legacy compatibility)")
	rootCmd.AddCommand(tickerCmd)
}

func runTicker(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}
	result, err := quoteSvc.GetRTTicker(cmdContext(cmd), sym.FutuCode(), service.RTTickerParams{
		Num:     tickerFlags.count,
		Periods: tickerFlags.periods,
	})
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderTickerTable(jsonObjectList(decodeJSONValue(result.Data), "ticker_list"))
	})
	return nil
}

func renderTickerTable(ticks []map[string]any) {
	tw := output.NewTableWriter(os.Stdout, "Time", "Price", "Volume", "Direction")
	for _, tick := range ticks {
		tw.AddRow(
			formatTimestamp(jsonInt(tick, "time")),
			formatPrice(jsonFloat(tick, "price")),
			fmt.Sprintf("%d", jsonInt(tick, "volume")),
			firstNonEmpty(jsonString(tick, "ticker_direction"), jsonString(tick, "direction")),
		)
	}
	tw.Render()
}
