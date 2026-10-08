package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Default values for the kline command flags.
const (
	defaultKLinePeriod = "day"
	defaultKLineCount  = 100
)

// Table header labels for the kline command.
const (
	klineHeaderTime   = "Time"
	klineHeaderOpen   = "Open"
	klineHeaderHigh   = "High"
	klineHeaderLow    = "Low"
	klineHeaderClose  = "Close"
	klineHeaderVolume = "Volume"
)

// klineFlags holds the flag values for the kline command.
var klineFlags struct {
	period string
	count  int
}

var klineCmd = &cobra.Command{
	Use:   "kline <symbol>",
	Short: "获取K线数据",
	Long:  "获取指定标的的K线（蜡烛图）数据，支持日线、周线、月线及多种分钟线。",
	Args:  cobra.ExactArgs(1),
	RunE:  runKLine,
}

func init() {
	klineCmd.Flags().StringVarP(&klineFlags.period, "period", "p", defaultKLinePeriod,
		"K线周期: day, week, month, 1m, 5m, 15m, 30m, 60m")
	klineCmd.Flags().IntVarP(&klineFlags.count, "count", "c", defaultKLineCount,
		"返回K线条数")
	rootCmd.AddCommand(klineCmd)
}

// runKLine is the top-level handler for the kline command.
func runKLine(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}

	ctx := cmdContext(cmd)
	bars, err := quoteSvc.GetKLine(ctx, sym.Full, klineFlags.period, klineFlags.count)
	if err != nil {
		return fmt.Errorf("fetching kline: %w", err)
	}

	printResult(bars, func() {
		renderKLineTable(bars)
	})
	return nil
}

// renderKLineTable writes K-line bar data as a formatted table to stdout.
func renderKLineTable(bars []service.KLine) {
	tw := output.NewTableWriter(os.Stdout,
		klineHeaderTime,
		klineHeaderOpen,
		klineHeaderHigh,
		klineHeaderLow,
		klineHeaderClose,
		klineHeaderVolume,
	)

	for _, b := range bars {
		tw.AddRow(
			formatTimestamp(b.Time),
			fmt.Sprintf("%.3f", b.Open),
			fmt.Sprintf("%.3f", b.High),
			fmt.Sprintf("%.3f", b.Low),
			fmt.Sprintf("%.3f", b.Close),
			fmt.Sprintf("%d", b.Volume),
		)
	}
	tw.Render()
}
