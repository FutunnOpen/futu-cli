package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Table header labels for the depth command.
const (
	depthHeaderAskPrice = "AskPrice"
	depthHeaderAskVol   = "AskVol"
	depthHeaderBidPrice = "BidPrice"
	depthHeaderBidVol   = "BidVol"
)

// Placeholder for missing depth levels when asks and bids differ in length.
const depthEmptyPlaceholder = "-"

var depthCmd = &cobra.Command{
	Use:   "depth <symbol>",
	Short: "获取买卖盘深度",
	Long:  "获取指定标的的买卖盘（order book）深度数据，逐档展示卖盘和买盘。",
	Args:  cobra.ExactArgs(1),
	RunE:  runDepth,
}

func init() {
	rootCmd.AddCommand(depthCmd)
}

// runDepth is the top-level handler for the depth command.
func runDepth(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}

	ctx := cmdContext(cmd)
	depth, err := quoteSvc.GetDepth(ctx, sym.Full)
	if err != nil {
		return fmt.Errorf("fetching depth: %w", err)
	}

	printResult(depth, func() {
		renderDepthTable(depth)
	})
	return nil
}

// renderDepthTable writes order book data as a formatted table to stdout,
// aligning asks and bids side-by-side.
func renderDepthTable(depth *service.Depth) {
	tw := output.NewTableWriter(os.Stdout,
		depthHeaderAskPrice,
		depthHeaderAskVol,
		depthHeaderBidPrice,
		depthHeaderBidVol,
	)

	rowCount := max(len(depth.Asks), len(depth.Bids))
	for i := 0; i < rowCount; i++ {
		askPrice, askVol := formatDepthEntry(depth.Asks, i)
		bidPrice, bidVol := formatDepthEntry(depth.Bids, i)
		tw.AddRow(askPrice, askVol, bidPrice, bidVol)
	}
	tw.Render()
}

// formatDepthEntry returns formatted price and volume strings for the entry
// at index i, or placeholder strings if the index is out of range.
func formatDepthEntry(entries []service.DepthEntry, i int) (string, string) {
	if i >= len(entries) {
		return depthEmptyPlaceholder, depthEmptyPlaceholder
	}
	return fmt.Sprintf("%.3f", entries[i].Price), fmt.Sprintf("%d", entries[i].Volume)
}
