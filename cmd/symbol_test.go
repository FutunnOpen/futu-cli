package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/service"
)

func TestParseFutuCodesUsesMarketCodeFormat(t *testing.T) {
	got, err := parseFutuCodes([]string{"us.aapl", "HK.700"})
	if err != nil {
		t.Fatalf("parseFutuCodes() error = %v", err)
	}
	want := []string{"US.AAPL", "HK.00700"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("codes = %#v, want %#v", got, want)
	}
}

func TestParseFutuCodesRejectsLegacyFormats(t *testing.T) {
	for _, input := range []string{"AAPL", "AAPL.US", "700"} {
		if _, err := parseFutuCodes([]string{input}); err == nil {
			t.Fatalf("parseFutuCodes(%q) expected error", input)
		}
	}
}

func TestParseOptionalFutuCode(t *testing.T) {
	got, err := parseOptionalFutuCode("HK.700")
	if err != nil {
		t.Fatalf("parseOptionalFutuCode() error = %v", err)
	}
	if got != "HK.00700" {
		t.Fatalf("code = %q, want HK.00700", got)
	}
}

func TestReadRESTFlagNormalizesSymbolParams(t *testing.T) {
	command := &cobra.Command{}
	command.Flags().String("symbol", "HK.00700", "")
	if err := command.Flags().Set("symbol", "us.aapl"); err != nil {
		t.Fatalf("set symbol flag: %v", err)
	}

	got, ok, err := readRESTFlag(command, quoteRESTParam{Name: "symbol", Kind: "string", Required: true})
	if err != nil {
		t.Fatalf("readRESTFlag() error = %v", err)
	}
	if !ok || got != "US.AAPL" {
		t.Fatalf("symbol flag = %#v, ok=%v; want US.AAPL", got, ok)
	}
}

func TestQuoteRESTFlagNamesUseSymbolTerminology(t *testing.T) {
	tests := map[string]string{
		"stock_owner":     "symbol",
		"filter_security": "filter-symbol",
		"code_list":       "code-list",
	}
	for input, want := range tests {
		if got := flagName(input); got != want {
			t.Fatalf("flagName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildHistoryOrdersRequestNormalizesSymbol(t *testing.T) {
	oldFlags, oldCfg := orderFlags, cfg
	defer func() { orderFlags, cfg = oldFlags, oldCfg }()
	cfg = nil
	orderFlags.account = "123"
	orderFlags.market = service.DefaultTradingMarket
	orderFlags.symbol = "us.aapl"

	req, err := buildHistoryOrdersRequest()
	if err != nil {
		t.Fatalf("buildHistoryOrdersRequest() error = %v", err)
	}
	if req.Symbol != "US.AAPL" {
		t.Fatalf("symbol = %q, want US.AAPL", req.Symbol)
	}
}

func TestBuildHistoryDealsRequestNormalizesSymbol(t *testing.T) {
	command := &cobra.Command{}
	registerDealHistoryFlags(command)
	if err := command.Flags().Set("account", "123"); err != nil {
		t.Fatalf("set account flag: %v", err)
	}
	if err := command.Flags().Set("symbol", "hk.700"); err != nil {
		t.Fatalf("set symbol flag: %v", err)
	}

	req, err := buildHistoryDealsRequest(command)
	if err != nil {
		t.Fatalf("buildHistoryDealsRequest() error = %v", err)
	}
	if req.Symbol != "HK.00700" {
		t.Fatalf("symbol = %q, want HK.00700", req.Symbol)
	}
}
