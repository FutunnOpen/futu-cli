package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/futunn/futu/internal/client"
	"github.com/futunn/futu/internal/reporting"
	"github.com/futunn/futu/internal/service"
)

const (
	newsTestSourceID     = "720"
	newsTestReportSource = "7"
)

type newsTestReporter struct{ events []reporting.Event }

func (r *newsTestReporter) Report(event reporting.Event) { r.events = append(r.events, event) }

func TestNewsBySourceUsesSourceIDAndSharedReporting(t *testing.T) {
	args, internal, err := extractInternalInvocationArgs([]string{
		"news-by-source", "--source-id", newsTestSourceID, "--source", "skill",
	})
	if err != nil || internal.source != reportSourceModeSkill {
		t.Fatalf("internal arguments: %#v, %v", internal, err)
	}
	spec := findNewsBySourceSpec(t)
	server := httptest.NewServer(http.HandlerFunc(checkNewsBySourceRequest(t, spec.Path)))
	defer server.Close()
	reporter := &newsTestReporter{}
	previous := quoteSvc
	quoteSvc = service.NewQuoteService(client.New(server.URL, client.WithToken("test-token"),
		client.WithClientName("test-client"), client.WithReportSource(newsTestReportSource), client.WithRequestReporter(reporter)))
	defer func() { quoteSvc = previous }()
	command := newQuoteRESTCommand(spec)
	command.SetArgs(args[1:])
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(reporter.events) != 1 || reporter.events[0].ToolID != "news_by_source" {
		t.Fatalf("unexpected reporting events: %#v", reporter.events)
	}
}

func findNewsBySourceSpec(t *testing.T) *quoteRESTSpec {
	t.Helper()
	specs := loadQuoteRESTSpecs()
	for i := range specs {
		if specs[i].ID == "news.by_source" {
			return &specs[i]
		}
	}
	t.Fatal("missing news.by_source spec")
	return nil
}

func checkNewsBySourceRequest(t *testing.T, path string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path || r.URL.Query().Get("source_id") != newsTestSourceID || r.URL.Query().Has("source") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Client-Name") != "test-client" || r.Header.Get("X-Fhl-Report-Source") != newsTestReportSource {
			t.Error("expected authentication, client and report source headers")
		}
		fmt.Fprint(w, `{"ret_code":0,"data":{}}`)
	}
}
