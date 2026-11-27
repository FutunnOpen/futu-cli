package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/futunn/futu/internal/client"
	"github.com/futunn/futu/internal/service"
	"github.com/spf13/cobra"
)

func TestNewsSearchDefaultsAndFilters(t *testing.T) {
	queries := make(chan url.Values, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.Query()
		fmt.Fprint(w, `{"ret_code":0,"data":{"items":[]}}`)
	}))
	defer server.Close()

	previous := quoteSvc
	quoteSvc = service.NewQuoteService(client.New(server.URL))
	defer func() { quoteSvc = previous }()

	for _, args := range [][]string{
		{"AAPL", "--type", "news"},
		{"腾讯", "--type", "news", "--page-size", "1", "--news-type", "NOTICE", "--sort-type", "PV", "--source-ids", "12,770", "--sources", "富途资讯", "--lang", "zh-CN"},
	} {
		command := newNewsSearchTestCommand(t)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	defaults := <-queries
	if defaults.Get("keyword") != "AAPL" || defaults.Get("page_size") != "10" || defaults.Get("news_type") != "ALL" || defaults.Get("sort_type") != "TIME" {
		t.Fatalf("default query = %v", defaults)
	}
	filtered := <-queries
	for key, want := range map[string]string{"keyword": "腾讯", "page_size": "1", "news_type": "NOTICE", "sort_type": "PV", "source_ids": "12,770", "sources": "富途资讯", "lang": "zh-CN"} {
		if got := filtered.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func newNewsSearchTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	specs := loadQuoteRESTSpecs()
	byCmd := make(map[string]*quoteRESTSpec, len(specs))
	for i := range specs {
		byCmd[specs[i].Cmd] = &specs[i]
	}
	for _, aggregate := range quoteRESTAggregates {
		if aggregate.Cmd == "search" {
			return newQuoteRESTAggregateCommand(aggregate, byCmd)
		}
	}
	t.Fatal("search aggregate not found")
	return nil
}

func TestNewsSearchRejectsInvalidParameters(t *testing.T) {
	cases := [][]string{
		{"--type", "news"},
		{"--type", "news", "--keyword", ""},
		{strings.Repeat("a", newsKeywordMaxChars+1), "--type", "news"},
		{"AAPL", "--type", "news", "--page-size", "51"},
		{"AAPL", "--type", "news", "--news-type", "OTHER"},
		{"AAPL", "--type", "news", "--sort-type", "HEAT"},
		{"AAPL", "--type", "news", "--source-ids", "12,abc"},
		{"AAPL", "--type", "news", "--lang", "fr"},
	}
	for _, args := range cases {
		command := newNewsSearchTestCommand(t)
		command.SilenceUsage = true
		command.SilenceErrors = true
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
