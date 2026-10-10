package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

// Rank command defaults.
const (
	defaultRankMarket = "HK"
	defaultRankCount  = 20
	defaultRankType   = "volume"
)

// Rank table headers.
const (
	rankHeaderRank     = "Rank"
	rankHeaderSymbol   = "Symbol"
	rankHeaderName     = "Name"
	rankHeaderLast     = "Last"
	rankHeaderChange   = "Change%"
	rankHeaderVolume   = "Volume"
	rankHeaderTurnover = "Turnover"
)

// validRankTypes lists all accepted rank type arguments.
var validRankTypes = []string{"volume", "turnover", "gainers", "losers", "amplitude"}

// rankFlags holds the flag values for the rank command.
var rankFlags struct {
	market string
	count  int
}

var rankCmd = &cobra.Command{
	Use:     "rank [type]",
	GroupID: quoteGroupID,
	Short:   "Show market rankings",
	Long:    "Show market rankings. type: volume, turnover, gainers, losers, amplitude. Defaults to the top 20 HK stocks by volume.",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runRank,
}

func init() {
	rankCmd.Flags().StringVarP(&rankFlags.market, "market", "m", defaultRankMarket,
		"Market (default: HK). HK / US / CN")
	rankCmd.Flags().IntVarP(&rankFlags.count, "count", "c", defaultRankCount,
		"Number of rows to return (default: 20)")
	rootCmd.AddCommand(rankCmd)
}

// runRank validates the rank type, fetches the ranking list, and renders the output.
func runRank(cmd *cobra.Command, args []string) error {
	rankType := defaultRankType
	if len(args) > 0 {
		rankType = args[0]
	}
	if err := validateRankType(rankType); err != nil {
		return err
	}

	ctx := cmdContext(cmd)
	items, err := quoteSvc.GetRank(ctx, rankType, rankFlags.market, rankFlags.count)
	if err != nil {
		return fmt.Errorf("get rank: %w", err)
	}

	printResult(items, func() {
		renderRankTable(items)
	})
	return nil
}

// validateRankType checks whether the given type is one of the accepted rank types.
func validateRankType(rankType string) error {
	for _, valid := range validRankTypes {
		if rankType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid rank type %q; valid types: %v", rankType, validRankTypes)
}

// renderRankTable writes the ranking items as an aligned table to stdout.
func renderRankTable(items []service.RankItem) {
	tw := output.NewTableWriter(os.Stdout,
		rankHeaderRank, rankHeaderSymbol, rankHeaderName,
		rankHeaderLast, rankHeaderChange, rankHeaderVolume, rankHeaderTurnover,
	)
	for i, item := range items {
		tw.AddRow(
			fmt.Sprintf("%d", i+1),
			item.Symbol,
			item.Name,
			fmt.Sprintf("%.3f", item.LastPrice),
			fmt.Sprintf("%.2f%%", item.ChangePct),
			fmt.Sprintf("%d", item.Volume),
			fmt.Sprintf("%.0f", item.Turnover),
		)
	}
	tw.Render()
}
