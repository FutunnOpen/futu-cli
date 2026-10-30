package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/service"
)

const (
	defaultUnusualTimeRange  = 7
	defaultUnusualLanguageID = 0
)

var unusualCmd = &cobra.Command{
	Use:     "unusual",
	Aliases: []string{"anomaly"},
	Short:   "Stock unusual-move analysis (SkillWrap)",
	Long:    "Query technical signal alerts, unusual capital flows, and unusual derivatives activity (OpenD SkillWrap APIs).",
}

var unusualTechnicalCmd = &cobra.Command{
	Use:     "technical-signals",
	Aliases: []string{"technical"},
	Short:   "Technical signal alerts (get_technical_unusual)",
	Args:    cobra.ExactArgs(1),
	RunE:    runUnusualTechnical,
}

var unusualCapitalCmd = &cobra.Command{
	Use:     "capital-flows",
	Aliases: []string{"capital"},
	Short:   "Unusual capital flows (get_financial_unusual)",
	Args:    cobra.ExactArgs(1),
	RunE:    runUnusualCapital,
}

var unusualDerivativesCmd = &cobra.Command{
	Use:     "derivatives-activity",
	Aliases: []string{"derivatives"},
	Short:   "Unusual derivatives activity (get_derivative_unusual)",
	Args:    cobra.ExactArgs(1),
	RunE:    runUnusualDerivatives,
}

func init() {
	registerUnusualFlags(unusualTechnicalCmd, true, false)
	registerUnusualFlags(unusualCapitalCmd, false, true)
	registerUnusualFlags(unusualDerivativesCmd, false, true)

	unusualCmd.AddCommand(unusualTechnicalCmd, unusualCapitalCmd, unusualDerivativesCmd)
	rootCmd.AddCommand(unusualCmd)
}

func registerUnusualFlags(cmd *cobra.Command, technical, dimensions bool) {
	cmd.Flags().Int("time-range", defaultUnusualTimeRange, "Natural-day lookback window")
	cmd.Flags().Int("language-id", defaultUnusualLanguageID, "Language: 0 zh-CN, 1 zh-TW, 2 en, 4 th, 5 ja")
	if technical {
		cmd.Flags().StringSlice("indicator-filters", nil, "Indicator filters (repeat flag or comma-separated)")
	}
	if dimensions {
		cmd.Flags().StringSlice("analysis-dimensions", nil, "Analysis dimensions (repeat flag or comma-separated)")
	}
}

func runUnusualTechnical(cmd *cobra.Command, args []string) error {
	return runUnusual(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetTechnicalUnusual(ctx, req)
	})
}

func runUnusualCapital(cmd *cobra.Command, args []string) error {
	return runUnusual(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetFinancialUnusual(ctx, req)
	})
}

func runUnusualDerivatives(cmd *cobra.Command, args []string) error {
	return runUnusual(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetDerivativeUnusual(ctx, req)
	})
}

type unusualFetcher func(context.Context, service.UnusualRequest) (*service.UnusualResult, error)

func runUnusual(cmd *cobra.Command, symbol string, fetch unusualFetcher) error {
	req, err := buildUnusualRequest(cmd, symbol)
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := fetch(ctx, req)
	if err != nil {
		return err
	}
	printResult(result, func() {
		fmt.Println(result.Content)
	})
	return nil
}

func buildUnusualRequest(cmd *cobra.Command, symbol string) (service.UnusualRequest, error) {
	timeRange, err := cmd.Flags().GetInt("time-range")
	if err != nil {
		return service.UnusualRequest{}, fmt.Errorf("reading time-range: %w", err)
	}
	languageID, err := cmd.Flags().GetInt("language-id")
	if err != nil {
		return service.UnusualRequest{}, fmt.Errorf("reading language-id: %w", err)
	}
	req := service.UnusualRequest{
		StockSymbol: strings.TrimSpace(symbol),
		TimeRange:   timeRange,
		LanguageID:  languageID,
	}
	if cmd.Flags().Lookup("indicator-filters") != nil {
		filters, err := expandRepeatedStringFlag(cmd, "indicator-filters")
		if err != nil {
			return service.UnusualRequest{}, err
		}
		req.IndicatorFilters = filters
	}
	if cmd.Flags().Lookup("analysis-dimensions") != nil {
		dims, err := expandRepeatedStringFlag(cmd, "analysis-dimensions")
		if err != nil {
			return service.UnusualRequest{}, err
		}
		req.AnalysisDimensions = dims
	}
	return req, nil
}

func expandRepeatedStringFlag(cmd *cobra.Command, name string) ([]string, error) {
	parts, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	var out []string
	for _, part := range parts {
		for _, item := range strings.Split(part, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out, nil
}
