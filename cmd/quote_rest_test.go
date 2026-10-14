package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestQuoteRESTAggregateCommandsReplaceFlatQuoteCommands(t *testing.T) {
	for _, name := range []string{
		"company",
		"corporate-actions",
		"financials",
		"ipo-list",
		"option-data",
		"shareholders",
		"valuation",
	} {
		if _, _, err := rootCmd.Find([]string{name}); err != nil {
			t.Fatalf("aggregate command %q not registered: %v", name, err)
		}
	}

	for _, name := range []string{
		"company-profile",
		"corporate-actions-dividends",
		"financials-statements",
		"ipo-list-us",
		"option-expiration-date",
		"shareholders-overview",
		"valuation-detail",
	} {
		if _, _, err := rootCmd.Find([]string{name}); err == nil {
			t.Fatalf("flat quote command %q is still registered", name)
		}
	}
}

func TestQuoteRESTAggregateSelectsVariantSpec(t *testing.T) {
	specs := loadQuoteRESTSpecs()
	specByCmd := make(map[string]*quoteRESTSpec, len(specs))
	for i := range specs {
		specByCmd[specs[i].Cmd] = &specs[i]
	}

	aggregate := quoteRESTAggregate{
		Cmd:             "ipo-list",
		Use:             "ipo-list",
		SelectorFlag:    "market",
		SelectorDefault: "HK",
		Variants: []quoteRESTVariant{
			{Value: "HK", Cmd: "ipo-list-hk"},
			{Value: "US", Cmd: "ipo-list-us"},
		},
	}
	command := newQuoteRESTAggregateCommand(aggregate, specByCmd)
	if command == nil {
		t.Fatal("aggregate command was nil")
	}
	if err := command.Flags().Set("market", "us"); err != nil {
		t.Fatalf("set market: %v", err)
	}
	selected, err := selectedQuoteRESTSpec(command, aggregate, map[string]*quoteRESTSpec{
		"HK": specByCmd["ipo-list-hk"],
		"US": specByCmd["ipo-list-us"],
	})
	if err != nil {
		t.Fatalf("selected spec: %v", err)
	}
	if selected.ID != "quote.ipo_list_us" {
		t.Fatalf("selected ID = %q, want quote.ipo_list_us", selected.ID)
	}
}

func TestSearchAggregateUsesKeywordFlagForSymbolQuery(t *testing.T) {
	specs := loadQuoteRESTSpecs()
	specByCmd := make(map[string]*quoteRESTSpec, len(specs))
	for i := range specs {
		specByCmd[specs[i].Cmd] = &specs[i]
	}
	var searchAggregate quoteRESTAggregate
	for _, aggregate := range quoteRESTAggregates {
		if aggregate.Cmd == "search" {
			searchAggregate = aggregate
			break
		}
	}
	command := newQuoteRESTAggregateCommand(searchAggregate, specByCmd)
	if command == nil {
		t.Fatal("search aggregate command was nil")
	}
	if command.Flags().Lookup("symbol") != nil {
		t.Fatal("search command should not expose --symbol")
	}
	flag := command.Flags().Lookup("keyword")
	if flag == nil {
		t.Fatal("search command should expose --keyword")
	}
	if !strings.Contains(flag.Usage, "keyword") || strings.Contains(flag.Usage, "MARKET.CODE") {
		t.Fatalf("keyword usage = %q", flag.Usage)
	}
}

func TestReadRESTFlagUsesSpecDefaultWithEmptyCommandDefault(t *testing.T) {
	command := &cobra.Command{}
	command.Flags().Int("limit", 0, "")

	got, ok, err := readRESTFlag(command, quoteRESTParam{
		Name:    "limit",
		Kind:    "int",
		Default: "30",
	})
	if err != nil {
		t.Fatalf("read flag: %v", err)
	}
	if !ok || got != 30 {
		t.Fatalf("limit = %#v, ok=%v; want 30", got, ok)
	}
}

func TestCLINameSymbolParamDoesNotNormalizeAsMarketCode(t *testing.T) {
	got, ok, err := parseRESTFlagString("keyword", quoteRESTParam{
		Name:    "symbol",
		CLIName: "keyword",
		Kind:    "string",
	}, "腾讯")
	if err != nil {
		t.Fatalf("parseRESTFlagString() error = %v", err)
	}
	if !ok || got != "腾讯" {
		t.Fatalf("keyword = %#v, ok=%v; want 腾讯", got, ok)
	}
}

func TestReadRESTFlagParsesStringRegisteredConflictFlag(t *testing.T) {
	command := &cobra.Command{}
	command.Flags().String("sort-type", "", "")
	if err := command.Flags().Set("sort-type", "2"); err != nil {
		t.Fatalf("set sort-type: %v", err)
	}

	got, ok, err := readRESTFlag(command, quoteRESTParam{
		Name: "sort_type",
		Kind: "int",
	})
	if err != nil {
		t.Fatalf("read flag: %v", err)
	}
	if !ok || got != 2 {
		t.Fatalf("sort-type = %#v, ok=%v; want 2", got, ok)
	}
}
