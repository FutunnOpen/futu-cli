package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/pkg/symbol"
)

const defaultRTDataSection = "NORMAL"

var rtDataFlags struct {
	section string
}

var rtDataCmd = &cobra.Command{
	Use:     "rt-data <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get intraday time-share data",
	Long:    "Get intraday time-share data for the symbol, grouped by trading session.",
	Example: `  futu rt-data HK.00700
  futu rt-data US.AAPL --section PREMARKET --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runRTData,
}

func init() {
	rtDataCmd.Flags().StringVar(&rtDataFlags.section, "section", defaultRTDataSection,
		"Trading session (default: NORMAL). NORMAL / FULL / PREMARKET / AFTERHOURS / HK_DARK / OVERNIGHT")
	rootCmd.AddCommand(rtDataCmd)
}

func runRTData(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}
	result, err := quoteSvc.GetRTData(cmdContext(cmd), sym.FutuCode(), rtDataFlags.section)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderRTDataTable(jsonObjectList(decodeJSONValue(result.Data), "section_list"))
	})
	return nil
}

func renderRTDataTable(sections []map[string]any) {
	tw := output.NewTableWriter(os.Stdout, "Section", "Time", "Price", "High", "Low", "Volume")
	for _, section := range sections {
		sectionName := jsonString(section, "trade_section")
		points := anyListToObjects(asAnyList(section["point_list"]))
		for _, point := range points {
			tw.AddRow(
				sectionName,
				formatTimestamp(jsonInt(point, "time")),
				formatPrice(jsonFloat(point, "cur_price")),
				formatPrice(jsonFloat(point, "high")),
				formatPrice(jsonFloat(point, "low")),
				fmt.Sprintf("%d", jsonInt(point, "volume")),
			)
		}
	}
	tw.Render()
}
