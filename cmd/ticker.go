package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Default value for the ticker command count flag.
const defaultTickerCount = 50

// Table header labels for the ticker command.
const (
	tickerHeaderTime      = "Time"
	tickerHeaderPrice     = "Price"
	tickerHeaderVolume    = "Volume"
	tickerHeaderDirection = "Direction"
)

// tickerFlags holds the flag values for the ticker command.
var tickerFlags struct {
	count int
}

var tickerCmd = &cobra.Command{
	Use:   "ticker <symbol>",
	Short: "获取逐笔成交",
	Long:  "获取指定标的的逐笔成交（tick-by-tick）数据，包含成交价、成交量和方向。",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicker,
}

func init() {
	tickerCmd.Flags().IntVarP(&tickerFlags.count, "count", "c", defaultTickerCount,
		"返回逐笔成交条数")
	rootCmd.AddCommand(tickerCmd)
}

// runTicker is the top-level handler for the ticker command.
func runTicker(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}

	ctx := cmdContext(cmd)
	ticks, err := quoteSvc.GetTicker(ctx, sym.Full, tickerFlags.count)
	if err != nil {
		return fmt.Errorf("fetching ticker: %w", err)
	}

	printResult(ticks, func() {
		renderTickerTable(ticks)
	})
	return nil
}

// renderTickerTable writes tick-by-tick trade data as a formatted table
// to stdout.
func renderTickerTable(ticks []service.Ticker) {
	tw := output.NewTableWriter(os.Stdout,
		tickerHeaderTime,
		tickerHeaderPrice,
		tickerHeaderVolume,
		tickerHeaderDirection,
	)

	for _, t := range ticks {
		tw.AddRow(
			formatTimestamp(t.Time),
			fmt.Sprintf("%.3f", t.Price),
			fmt.Sprintf("%d", t.Volume),
			t.Direction,
		)
	}
	tw.Render()
}
