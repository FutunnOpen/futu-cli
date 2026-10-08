package reporting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMatchToolRouteUsesMethodAndTemplate(t *testing.T) {
	postRoute := MatchToolRoute(http.MethodPost, "/api/v1.0/accounts/123/orders")
	if postRoute.ID != "trading_order_place" {
		t.Fatalf("POST route = %q, want trading_order_place", postRoute.ID)
	}

	getRoute := MatchToolRoute(http.MethodGet, "/api/v1.0/accounts/123/orders")
	if getRoute.ID != "account_orders_active" {
		t.Fatalf("GET route = %q, want account_orders_active", getRoute.ID)
	}
}

func TestExtractOrderIDPriority(t *testing.T) {
	got := ExtractOrderID(
		[]byte(`{"data":{"order_id":"from-response"}}`),
		[]byte(`{"order_id":"from-request"}`),
		"/api/v1.0/accounts/1/orders/from-path",
		"/api/v1.0/accounts/{acc_id}/orders/{order_id}",
		"order_id=from-query",
	)
	if got != "from-response" {
		t.Fatalf("order id = %q, want from-response", got)
	}
}

func TestExtractOrderIDFallbacksToPath(t *testing.T) {
	got := ExtractOrderID(nil, nil,
		"/api/v1/crypto/accounts/1/orders/ABC",
		"/api/v1/crypto/accounts/{acc_id}/orders/{orderId}",
		"",
	)
	if got != "ABC" {
		t.Fatalf("order id = %q, want ABC", got)
	}
}

func TestSanitizeCoarseKeepsOrderIDOnly(t *testing.T) {
	got := SanitizeBody([]byte(`{"order_id":"OID","price":"123","nested":{"token":"secret","qty":10}}`), true)
	if !strings.Contains(got, `"order_id":"OID"`) {
		t.Fatalf("sanitized body should keep order_id: %s", got)
	}
	if strings.Contains(got, "123") || strings.Contains(got, "secret") || strings.Contains(got, "10") {
		t.Fatalf("sanitized body leaked sensitive values: %s", got)
	}
}

func TestSanitizeNonCoarseOnlyRedactsCredentialFields(t *testing.T) {
	got := SanitizeQuery("symbol=US.AAPL&token=secret", false)
	if !strings.Contains(got, `"symbol":"US.AAPL"`) {
		t.Fatalf("sanitized query should keep ordinary field: %s", got)
	}
	if strings.Contains(got, "secret") {
		t.Fatalf("sanitized query leaked token: %s", got)
	}
}

func TestBuildEventGeneratesUniqueTraceID(t *testing.T) {
	first := BuildEvent(BuildInput{Source: "1", Method: http.MethodGet, APIPath: "/v1/quote/kline"})
	second := BuildEvent(BuildInput{Source: "1", Method: http.MethodGet, APIPath: "/v1/quote/kline"})
	if first.TraceID == "" || second.TraceID == "" || first.TraceID == second.TraceID {
		t.Fatalf("trace ids should be non-empty and unique: %q %q", first.TraceID, second.TraceID)
	}
	if first.OrderID != defaultOrderID {
		t.Fatalf("order id = %q, want %q", first.OrderID, defaultOrderID)
	}
	if first.ToolID != unknownToolID {
		t.Fatalf("tool id = %q, want %q", first.ToolID, unknownToolID)
	}
}

func TestReporterPostsEventWithBearerToken(t *testing.T) {
	received := make(chan reportToolCallRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get(headerAuthorization); auth != "Bearer token-1" {
			t.Fatalf("Authorization = %q, want Bearer token-1", auth)
		}
		var req reportToolCallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		received <- req
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reporter := NewReporter(Options{
		Enabled:  true,
		Endpoint: srv.URL,
		Token:    "token-1",
	})
	reporter.Report(Event{TraceID: "trace-1", ToolID: "tool-1"})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := reporter.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	select {
	case req := <-received:
		if req.Event.TraceID != "trace-1" || req.Event.ToolID != "tool-1" {
			t.Fatalf("event = %+v", req.Event)
		}
	default:
		t.Fatal("expected report request")
	}
}
