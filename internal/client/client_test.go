package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FutunnOpen/futu-cli/internal/reporting"
)

// ── OAuth mode headers ──────────────────────────────────────────────────────

func TestSetHeaders_OAuthMode(t *testing.T) {
	c := New("https://api.example.com",
		WithToken("my-token"),
		WithClientName("test-client"),
		WithVersion("1.0.0"),
	)

	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1/test", nil)
	c.setHeaders(req, context.Background(), nil)

	tests := []struct {
		header string
		want   string
	}{
		{headerUserAgent, "futu/1.0.0"},
		{headerClientName, "test-client"},
		{headerAuth, "Bearer my-token"},
	}
	for _, tt := range tests {
		got := req.Header.Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}

}

func TestSetHeaders_OAuthMode_NoToken(t *testing.T) {
	c := New("https://api.example.com",
		WithToken(""),
	)

	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1/test", nil)
	c.setHeaders(req, context.Background(), nil)

	if auth := req.Header.Get(headerAuth); len(auth) > 0 {
		t.Errorf("Authorization should be empty when token is empty, got %q", auth)
	}
}

func TestSetHeaders_ContentType_WithBody(t *testing.T) {
	c := New("https://api.example.com")
	req, _ := http.NewRequest(http.MethodPost, "https://api.example.com/v1/order", nil)
	c.setHeaders(req, context.Background(), []byte(`{"qty":1}`))

	if ct := req.Header.Get(headerContentType); ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
}

func TestSetHeaders_ContentType_NoBody(t *testing.T) {
	c := New("https://api.example.com")
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1/quote", nil)
	c.setHeaders(req, context.Background(), nil)

	if ct := req.Header.Get(headerContentType); len(ct) > 0 {
		t.Errorf("Content-Type should be absent for GET without body, got %q", ct)
	}
}

func TestSetHeaders_CliCmdFromContext(t *testing.T) {
	c := New("https://api.example.com")
	ctx := context.WithValue(context.Background(), CtxKeyCommand, "futu kline")
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1/kline", nil)
	c.setHeaders(req, ctx, nil)

	if cmd := req.Header.Get(headerCliCmd); cmd != "futu kline" {
		t.Errorf("X-Cli-Cmd = %q, want %q", cmd, "futu kline")
	}
}

func TestTradeMutationEndpointMatrixRequiresReportSource(t *testing.T) {
	mutatingEndpoints := []string{
		"trade.order.place",
		"trade.order.modify",
		"trade.order.cancel",
		"trade.order.confirm",
		"crypto.order.place",
		"crypto.order.modify",
		"crypto.order.cancel",
		"web3.swap.execute",
		"web3.transfer",
	}
	for _, endpoint := range mutatingEndpoints {
		if !Endpoints[endpoint].ReportSource {
			t.Fatalf("%s must require report source header", endpoint)
		}
	}

	readOnlyEndpoints := []string{
		"quote.snapshot",
		"trade.order.open",
		"crypto.order.active",
		"web3.swap.quote",
		"web3.wallet.balance",
	}
	for _, endpoint := range readOnlyEndpoints {
		if Endpoints[endpoint].ReportSource {
			t.Fatalf("%s must not require report source header", endpoint)
		}
	}
}

func TestCallAddsReportSourceForTradeMutations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(headerReportSource); got != "source-1" {
			t.Fatalf("%s = %q, want source-1", headerReportSource, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithReportSource("source-1"))
	err := c.Call(context.Background(), "trade.order.place", CallOpts{
		PathParams: map[string]string{"acc_id": "123"},
	}, nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

func TestCallOmitsReportSourceForReadOnlyEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(headerReportSource); len(got) > 0 {
			t.Fatalf("%s = %q, want empty", headerReportSource, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithReportSource("source-1"))
	if err := c.Call(context.Background(), "trade.order.open", CallOpts{
		PathParams: map[string]string{"acc_id": "123"},
	}, nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

// ── Token management ────────────────────────────────────────────────────────

func TestSetToken_ReadToken(t *testing.T) {
	c := New("https://api.example.com", WithToken("initial"))
	if tok := c.readToken(); tok != "initial" {
		t.Errorf("initial token = %q, want %q", tok, "initial")
	}

	c.SetToken("updated")
	if tok := c.readToken(); tok != "updated" {
		t.Errorf("updated token = %q, want %q", tok, "updated")
	}
}

// ── Constructor ─────────────────────────────────────────────────────────────

func TestNew_Defaults(t *testing.T) {
	c := New("https://api.example.com/")
	if c.base != "https://api.example.com" {
		t.Errorf("base = %q, want trailing slash trimmed", c.base)
	}
	if c.version != "dev" {
		t.Errorf("version = %q, want %q", c.version, "dev")
	}
}

// ── buildURL ────────────────────────────────────────────────────────────────

func TestBuildURL_NoQuery(t *testing.T) {
	c := New("https://api.example.com")
	got := c.buildURL("/v1/quote", "")
	if got != "https://api.example.com/v1/quote" {
		t.Errorf("url = %q", got)
	}
}

func TestBuildURL_WithQuery(t *testing.T) {
	c := New("https://api.example.com")
	got := c.buildURL("/v1/quote", "symbol=700.HK")
	if got != "https://api.example.com/v1/quote?symbol=700.HK" {
		t.Errorf("url = %q", got)
	}
}

// ── resolvePathParams ───────────────────────────────────────────────────────

func TestResolvePathParams(t *testing.T) {
	got := resolvePathParams("/v1/orders/{id}/cancel", map[string]string{"id": "12345"})
	if got != "/v1/orders/12345/cancel" {
		t.Errorf("got %q", got)
	}
}

func TestResolvePathParams_SpecialChars(t *testing.T) {
	got := resolvePathParams("/v1/{symbol}", map[string]string{"symbol": "BTC/USD"})
	if got != "/v1/BTC%2FUSD" {
		t.Errorf("got %q, expected URL-encoded slash", got)
	}
}

// ── End-to-end with httptest ────────────────────────────────────────────────

func TestDo_OAuthMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer token", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, WithToken("test-token"))

	var result map[string]string
	err := c.Do(context.Background(), http.MethodGet, "/v1/test", nil, nil, &result)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %q", result["status"])
	}
}

func TestDo_PostWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order_id":"123"}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	body := map[string]any{"symbol": "700.HK", "qty": 100}
	var result map[string]string
	err := c.Do(context.Background(), http.MethodPost, "/v1/order", nil, body, &result)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if result["order_id"] != "123" {
		t.Errorf("order_id = %q", result["order_id"])
	}
}

type capturedReportRequest struct {
	Event reporting.Event `json:"event"`
}

func TestCallReportsThroughAsyncReporter(t *testing.T) {
	received := make(chan capturedReportRequest, 1)
	reportSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer oauth-token" {
			t.Fatalf("Authorization = %q, want Bearer oauth-token", auth)
		}
		var req capturedReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode report request: %v", err)
		}
		received <- req
		w.WriteHeader(http.StatusOK)
	}))
	defer reportSrv.Close()

	businessSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"order_id":"OID-1","price":"123"}}`))
	}))
	defer businessSrv.Close()

	reporter := reporting.NewReporter(reporting.Options{
		Enabled:  true,
		Endpoint: reportSrv.URL,
		Token:    "oauth-token",
	})
	c := New(businessSrv.URL,
		WithToken("oauth-token"),
		WithReportSource("1"),
		WithReportMetadata("futu", "codex"),
		WithRequestReporter(reporter),
		WithVersion("v1.2.3"),
	)

	ctx := context.WithValue(context.Background(), CtxKeyCommand, "futu order place")
	var result map[string]any
	err := c.Call(ctx, "trade.order.place", CallOpts{
		PathParams: map[string]string{"acc_id": "123"},
		Body:       map[string]any{"symbol": "US.AAPL", "qty": 1},
	}, &result)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := reporter.Close(closeCtx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	select {
	case req := <-received:
		event := req.Event
		if event.Source != 1 || event.AgentName != "codex" || event.Platform != "futu" {
			t.Fatalf("unexpected source metadata: %+v", event)
		}
		if event.ToolID != "trading_order_place" {
			t.Fatalf("tool id = %q, want trading_order_place", event.ToolID)
		}
		if event.APIPath != "/api/v1.0/accounts/123/orders" {
			t.Fatalf("api path = %q", event.APIPath)
		}
		if event.APIPathTemplate != "/api/v1.0/accounts/{acc_id}/orders" {
			t.Fatalf("path template = %q", event.APIPathTemplate)
		}
		if event.OrderID != "OID-1" {
			t.Fatalf("order id = %q, want OID-1", event.OrderID)
		}
		if event.TraceID == "" {
			t.Fatal("trace id should not be empty")
		}
	default:
		t.Fatal("expected async report request")
	}
}

// ── marshalBody ─────────────────────────────────────────────────────────────

func TestMarshalBody_Nil(t *testing.T) {
	b, err := marshalBody(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

func TestMarshalBody_Struct(t *testing.T) {
	body := struct {
		Name string `json:"name"`
	}{Name: "test"}
	b, err := marshalBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != `{"name":"test"}` {
		t.Errorf("got %q", string(b))
	}
}

// ── commandFromCtx ──────────────────────────────────────────────────────────

func TestCommandFromCtx_Set(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyCommand, "futu quote")
	if cmd := commandFromCtx(ctx); cmd != "futu quote" {
		t.Errorf("got %q", cmd)
	}
}

func TestCommandFromCtx_NotSet(t *testing.T) {
	if cmd := commandFromCtx(context.Background()); cmd != "" {
		t.Errorf("got %q, want empty", cmd)
	}
}
