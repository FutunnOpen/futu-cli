package client

import (
	"strings"
	"testing"

	"github.com/futunn/futu/internal/reporting"
)

const newsSearchToolID = "news_search"

func TestQuoteEndpointsReportExpectedToolID(t *testing.T) {
	for endpoint, route := range Endpoints {
		if !strings.HasPrefix(endpoint, "quote.") && !strings.HasPrefix(endpoint, "skillwrap.") && !strings.HasPrefix(endpoint, "news.") {
			continue
		}
		t.Run(endpoint, func(t *testing.T) {
			want := strings.ReplaceAll(endpoint, ".", "_")
			if endpoint == "quote.news_search" {
				want = newsSearchToolID
			}
			path := resolvePathParams(route.Path, placeholderPathParams(route.Path))
			for _, input := range []reporting.BuildInput{
				{Method: route.Method, APIPath: path, APIPathTemplate: route.Path},
				{Method: route.Method, APIPath: path},
			} {
				event := reporting.BuildEvent(input)
				if event.ToolID != want {
					t.Errorf("%s %s: tool_id = %q, want %q", route.Method, path, event.ToolID, want)
				}
				if event.APIPathTemplate != route.Path {
					t.Errorf("%s %s: template = %q, want %q", route.Method, path, event.APIPathTemplate, route.Path)
				}
			}
		})
	}
}
