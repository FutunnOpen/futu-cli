package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReporterFlushPostsEvents(t *testing.T) {
	received := make(chan flushRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != contentTypeJSON {
			t.Fatalf("Content-Type = %q, want %q", got, contentTypeJSON)
		}
		var req flushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		received <- req
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	reporter := NewReporter(server.URL, true)
	reporter.Report(Event{Command: "quote", Outcome: "success"})
	reporter.Flush()

	select {
	case req := <-received:
		if len(req.Events) != 1 || req.Events[0].Command != "quote" {
			t.Fatalf("events = %#v", req.Events)
		}
	default:
		t.Fatal("expected analytics request")
	}
}
