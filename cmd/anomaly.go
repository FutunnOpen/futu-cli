package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/service"
)

const (
	defaultAnomalyTimeRange  = 7
	defaultAnomalyLanguageID = 0
)

var anomalyCmd = &cobra.Command{
	Use:   "anomaly",
	Short: "Stock unusual-move analysis (SkillWrap)",
	Long:  "Query technical, capital-flow, and derivatives unusual-move analysis aligned with OpenD SkillWrap APIs.",
}

var anomalyTechnicalCmd = &cobra.Command{
	Use:   "technical <symbol>",
	Short: "Technical unusual analysis",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnomalyTechnical,
}

var anomalyCapitalCmd = &cobra.Command{
	Use:   "capital <symbol>",
	Short: "Financial / capital unusual analysis",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnomalyCapital,
}

var anomalyDerivativesCmd = &cobra.Command{
	Use:   "derivatives <symbol>",
	Short: "Derivatives unusual analysis",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnomalyDerivatives,
}

func init() {
	registerAnomalyFlags(anomalyTechnicalCmd, true, false)
	registerAnomalyFlags(anomalyCapitalCmd, false, true)
	registerAnomalyFlags(anomalyDerivativesCmd, false, true)

	anomalyCmd.AddCommand(anomalyTechnicalCmd, anomalyCapitalCmd, anomalyDerivativesCmd)
	rootCmd.AddCommand(anomalyCmd)
}

func registerAnomalyFlags(cmd *cobra.Command, technical, dimensions bool) {
	cmd.Flags().Int("time-range", defaultAnomalyTimeRange, "Natural-day lookback window")
	cmd.Flags().Int("language-id", defaultAnomalyLanguageID, "Language: 0 zh-CN, 1 zh-TW, 2 en, 4 th, 5 ja")
	if technical {
		cmd.Flags().StringSlice("indicator-filters", nil, "Indicator filters (repeat flag or comma-separated)")
	}
	if dimensions {
		cmd.Flags().StringSlice("analysis-dimensions", nil, "Analysis dimensions (repeat flag or comma-separated)")
	}
}

func runAnomalyTechnical(cmd *cobra.Command, args []string) error {
	return runAnomaly(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetTechnicalUnusual(ctx, req)
	})
}

func runAnomalyCapital(cmd *cobra.Command, args []string) error {
	return runAnomaly(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetFinancialUnusual(ctx, req)
	})
}

func runAnomalyDerivatives(cmd *cobra.Command, args []string) error {
	return runAnomaly(cmd, args[0], func(ctx context.Context, req service.UnusualRequest) (*service.UnusualResult, error) {
		return skillWrapSvc.GetDerivativeUnusual(ctx, req)
	})
}

type unusualFetcher func(context.Context, service.UnusualRequest) (*service.UnusualResult, error)

func runAnomaly(cmd *cobra.Command, symbol string, fetch unusualFetcher) error {
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
