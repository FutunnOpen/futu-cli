package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestUnusualCommandsRegistered(t *testing.T) {
	cases := []struct {
		args []string
		use  string
	}{
		{[]string{"unusual", "technical-signals"}, "technical-signals"},
		{[]string{"anomaly", "technical"}, "technical-signals"},
		{[]string{"unusual", "capital-flows"}, "capital-flows"},
		{[]string{"anomaly", "capital"}, "capital-flows"},
		{[]string{"unusual", "derivatives-activity"}, "derivatives-activity"},
		{[]string{"anomaly", "derivatives"}, "derivatives-activity"},
	}
	for _, tc := range cases {
		cmd, _, err := rootCmd.Find(tc.args)
		if err != nil {
			t.Fatalf("find %v: %v", tc.args, err)
		}
		if cmd.Name() != tc.use {
			t.Fatalf("find %v name = %q, want %q", tc.args, cmd.Name(), tc.use)
		}
	}
}

func TestBuildUnusualTechnicalRequest(t *testing.T) {
	command := &cobra.Command{}
	registerUnusualFlags(command, true, false)
	if err := command.Flags().Set("time-range", "14"); err != nil {
		t.Fatalf("set time-range: %v", err)
	}
	if err := command.Flags().Set("language-id", "2"); err != nil {
		t.Fatalf("set language-id: %v", err)
	}
	if err := command.Flags().Set("indicator-filters", "MACD,RSI6"); err != nil {
		t.Fatalf("set indicator-filters: %v", err)
	}

	req, err := buildUnusualRequest(command, " HK.00700 ")
	if err != nil {
		t.Fatalf("buildUnusualRequest() error = %v", err)
	}
	if req.StockSymbol != "HK.00700" || req.TimeRange != 14 || req.LanguageID != 2 {
		t.Fatalf("request = %#v", req)
	}
	if len(req.IndicatorFilters) != 2 || req.IndicatorFilters[0] != "MACD" || req.IndicatorFilters[1] != "RSI6" {
		t.Fatalf("indicator filters = %#v", req.IndicatorFilters)
	}
	if len(req.AnalysisDimensions) != 0 {
		t.Fatalf("analysis dimensions = %#v", req.AnalysisDimensions)
	}
}

func TestBuildUnusualCapitalRequestSplitsDimensions(t *testing.T) {
	command := &cobra.Command{}
	registerUnusualFlags(command, false, true)
	if err := command.Flags().Set("analysis-dimensions", "funds_flow, funds_broker"); err != nil {
		t.Fatalf("set analysis-dimensions: %v", err)
	}

	req, err := buildUnusualRequest(command, "US.AAPL")
	if err != nil {
		t.Fatalf("buildUnusualRequest() error = %v", err)
	}
	if req.TimeRange != defaultUnusualTimeRange || req.LanguageID != defaultUnusualLanguageID {
		t.Fatalf("defaults = %#v", req)
	}
	if len(req.AnalysisDimensions) != 2 || req.AnalysisDimensions[0] != "funds_flow" || req.AnalysisDimensions[1] != "funds_broker" {
		t.Fatalf("analysis dimensions = %#v", req.AnalysisDimensions)
	}
	if len(req.IndicatorFilters) != 0 {
		t.Fatalf("indicator filters = %#v", req.IndicatorFilters)
	}
}
