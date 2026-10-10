package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/pkg/symbol"
)

const depthEmptyPlaceholder = "-"

var orderBookFlags struct {
	num int
}

var orderBookCmd = &cobra.Command{
	Use:     "order-book <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get order book",
	Long:    "Get the real-time order book for the specified symbol. Market-data permissions determine depth; --num only caps the returned levels.",
	Example: `  futu order-book HK.00700
  futu order-book US.AAPL --num 10 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runOrderBook,
}

var depthCmd = &cobra.Command{
	Use:     "depth <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get order book (alias of order-book)",
	Long:    "Alias of order-book.",
	Args:    cobra.ExactArgs(1),
	RunE:    runOrderBook,
}

func init() {
	orderBookCmd.Flags().IntVarP(&orderBookFlags.num, "num", "n", 0,
		"Maximum order-book depth (default: 0, return the maximum allowed by market-data permissions; range 1-60)")
	depthCmd.Flags().IntVarP(&orderBookFlags.num, "num", "n", 0,
		"Maximum order-book depth (default: 0, return the maximum allowed by market-data permissions; range 1-60)")
	rootCmd.AddCommand(orderBookCmd, depthCmd)
}

func runOrderBook(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}
	result, err := quoteSvc.GetOrderBook(
		cmdContext(cmd),
		sym.FutuCode(),
		orderBookFlags.num,
		cmd.Flags().Changed("num"),
	)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderOrderBookTable(decodeJSONValue(result.Data))
	})
	return nil
}

func renderOrderBookTable(data any) {
	items := anyListToObjects(asAnyList(data))
	if len(items) == 0 {
		return
	}
	books := anyListToObjects(asAnyList(items[0]["books"]))
	if len(books) == 0 {
		return
	}
	tw := output.NewTableWriter(os.Stdout, "Exchange", "AskPrice", "AskVol", "BidPrice", "BidVol")
	for _, book := range books {
		bids := anyListToObjects(asAnyList(book["bid_list"]))
		asks := anyListToObjects(asAnyList(book["ask_list"]))
		exchange := jsonString(book, "exchange")
		rowCount := max(len(asks), len(bids))
		for i := 0; i < rowCount; i++ {
			askPrice, askVol := formatBookLevel(asks, i)
			bidPrice, bidVol := formatBookLevel(bids, i)
			tw.AddRow(exchange, askPrice, askVol, bidPrice, bidVol)
		}
	}
	tw.Render()
}

func asAnyList(value any) []any {
	list, _ := value.([]any)
	return list
}

func formatBookLevel(levels []map[string]any, i int) (string, string) {
	if i >= len(levels) {
		return depthEmptyPlaceholder, depthEmptyPlaceholder
	}
	return formatPrice(jsonFloat(levels[i], "price")), fmt.Sprintf("%d", jsonInt(levels[i], "volume"))
}
