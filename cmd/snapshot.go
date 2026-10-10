package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
)

var snapshotCmd = &cobra.Command{
	Use:     "snapshot <symbol...>",
	GroupID: quoteGroupID,
	Short:   "Get market snapshots",
	Long:    "Get market snapshots in batch, up to 400 symbols per request.",
	Example: `  futu snapshot HK.00700
  futu snapshot HK.00700 US.AAPL --format json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSnapshot,
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
}

func runSnapshot(cmd *cobra.Command, args []string) error {
	codes, err := parseFutuCodes(args)
	if err != nil {
		return err
	}
	result, err := quoteSvc.GetMarketSnapshot(cmdContext(cmd), codes)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderSnapshotTable(jsonObjectList(decodeJSONValue(result.Data), "snapshot_list"))
	})
	return nil
}

func renderSnapshotTable(items []map[string]any) {
	tw := output.NewTableWriter(os.Stdout,
		"Code", "Name", "Last", "Change%", "High", "Low", "Volume", "Turnover",
	)
	for _, item := range items {
		last := jsonFloat(item, "last_price")
		prev := jsonFloat(item, "prev_close_price")
		var changePct float64
		if prev != 0 {
			changePct = (last - prev) / prev * 100
		}
		tw.AddRow(
			jsonString(item, "code"),
			firstNonEmpty(jsonString(item, "sc_name"), jsonString(item, "name")),
			formatPrice(last),
			formatChangePct(changePct),
			formatPrice(jsonFloat(item, "high_price")),
			formatPrice(jsonFloat(item, "low_price")),
			fmt.Sprintf("%d", jsonInt(item, "volume")),
			formatTurnover(jsonFloat(item, "turnover")),
		)
	}
	tw.Render()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return ""
}
