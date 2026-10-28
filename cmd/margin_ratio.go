package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

var marginRatioCmd = &cobra.Command{
	Use:   "margin-ratio",
	Short: "Query margin and short-selling ratios",
	Long:  "Query margin ratios for up to 10 symbols. Requires --security-firm and --codes.",
	Args:  cobra.NoArgs,
	RunE:  runMarginRatio,
}

func init() {
	marginRatioCmd.Flags().String("security-firm", "", "Security firm code (e.g. FUTUSECURITIES, FUTUINC)")
	marginRatioCmd.Flags().String("codes", "", "Comma-separated symbols (MARKET.CODE)")
	_ = marginRatioCmd.MarkFlagRequired("security-firm")
	_ = marginRatioCmd.MarkFlagRequired("codes")
	rootCmd.AddCommand(marginRatioCmd)
}

func runMarginRatio(cmd *cobra.Command, _ []string) error {
	securityFirm, err := cmd.Flags().GetString("security-firm")
	if err != nil {
		return fmt.Errorf("reading security-firm: %w", err)
	}
	codes, err := cmd.Flags().GetString("codes")
	if err != nil {
		return fmt.Errorf("reading codes: %w", err)
	}
	ctx := cmdContext(cmd)
	ratios, err := tradeSvc.GetMarginRatio(ctx, service.MarginRatioRequest{
		SecurityFirm: securityFirm,
		Codes:        codes,
	})
	if err != nil {
		return fmt.Errorf("fetching margin ratio: %w", err)
	}
	printResult(ratios, func() {
		renderMarginRatioTable(ratios)
	})
	return nil
}

func renderMarginRatioTable(ratios []service.MarginRatio) {
	tw := output.NewTableWriter(os.Stdout,
		"Code", "Long", "Short", "IM Long", "IM Short", "Short Fee", "Short Pool",
	)
	for _, r := range ratios {
		tw.AddRow(
			r.Code,
			formatBoolPermit(r.IsLongPermit),
			formatBoolPermit(r.IsShortPermit),
			r.IMLongRatio,
			r.IMShortRatio,
			r.ShortFeeRate,
			r.ShortPoolRemain,
		)
	}
	tw.Render()
}

func formatBoolPermit(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
