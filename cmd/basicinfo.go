package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
)

var basicInfoCmd = &cobra.Command{
	Use:     "basicinfo <symbol...>",
	GroupID: quoteGroupID,
	Short:   "Get basic symbol information",
	Long:    "Get static symbol information in batch, excluding quotes. Up to 400 symbols per request.",
	Example: `  futu basicinfo HK.00700
  futu basicinfo HK.00700 US.AAPL --format json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runBasicInfo,
}

func init() {
	rootCmd.AddCommand(basicInfoCmd)
}

func runBasicInfo(cmd *cobra.Command, args []string) error {
	codes, err := parseFutuCodes(args)
	if err != nil {
		return err
	}
	result, err := quoteSvc.GetStockBasicInfo(cmdContext(cmd), codes)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderBasicInfoTable(jsonObjectList(decodeJSONValue(result.Data), "basic_list"))
	})
	return nil
}

func renderBasicInfoTable(items []map[string]any) {
	tw := output.NewTableWriter(os.Stdout, "Code", "Name", "Type", "Lot", "Exchange", "Suspended")
	for _, item := range items {
		tw.AddRow(
			jsonString(item, "code"),
			firstNonEmpty(jsonString(item, "sc_name"), jsonString(item, "name")),
			jsonString(item, "stock_type"),
			fmt.Sprintf("%d", jsonInt(item, "lot_size")),
			jsonString(item, "exchange"),
			fmt.Sprintf("%v", item["suspension"]),
		)
	}
	tw.Render()
}
