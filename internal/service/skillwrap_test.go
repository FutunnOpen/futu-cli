package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/futunn/futu/internal/client"
)

func TestGetTechnicalUnusualUsesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1.0/quote/HK.00700/unusual/technical" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.ContentLength > 0 {
			t.Fatalf("GET request must not send a body")
		}
		query := r.URL.Query()
		if query.Get("time_range") != "7" || query.Get("language_id") != "0" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		if query.Get("stock_symbol") != "" {
			t.Fatalf("symbol must be a path parameter, query = %s", r.URL.RawQuery)
		}
		if got := query["indicator_filters"]; len(got) != 1 || got[0] != "MACD" {
			t.Fatalf("indicator_filters = %#v", got)
		}
		fmt.Fprint(w, `{"ret_code":0,"ret_msg":"success","data":{"code":0,"message":"","time_range":"2026.1.1-2026.1.8","content":"ok"}}`)
	}))
	defer server.Close()

	svc := NewSkillWrapService(client.New(server.URL))
	result, err := svc.GetTechnicalUnusual(context.Background(), UnusualRequest{
		StockSymbol:      "HK.00700",
		IndicatorFilters: []string{"MACD"},
	})
	if err != nil {
		t.Fatalf("GetTechnicalUnusual() error = %v", err)
	}
	if result.Content != "ok" || result.Method != "get_technical_unusual" || result.ErrCode != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestGetFinancialUnusualNoAnomaly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1.0/quote/US.AAPL/unusual/capital" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"ret_code":0,"ret_msg":"success","data":{"code":1,"message":"","time_range":"","content":""}}`)
	}))
	defer server.Close()

	svc := NewSkillWrapService(client.New(server.URL))
	result, err := svc.GetFinancialUnusual(context.Background(), UnusualRequest{StockSymbol: "US.AAPL"})
	if err != nil {
		t.Fatalf("GetFinancialUnusual() error = %v", err)
	}
	if result.ErrCode != 1 || result.Content != "" || result.RetType != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestGetDerivativeUnusualQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		query := r.URL.Query()["analysis_dimensions"]
		if len(query) != 2 || query[0] != "option_unusual" || query[1] != "option_volatility" {
			t.Fatalf("analysis_dimensions = %#v", query)
		}
		if queryTime := r.URL.Query().Get("time_range"); queryTime != "30" {
			t.Fatalf("time_range = %s", queryTime)
		}
		fmt.Fprint(w, `{"ret_code":0,"ret_msg":"success","data":{"code":0,"message":"","time_range":"近30个自然日","content":"options"}}`)
	}))
	defer server.Close()

	svc := NewSkillWrapService(client.New(server.URL))
	result, err := svc.GetDerivativeUnusual(context.Background(), UnusualRequest{
		StockSymbol:        "US.AAPL",
		TimeRange:          30,
		AnalysisDimensions: []string{"option_unusual", "option_volatility"},
	})
	if err != nil {
		t.Fatalf("GetDerivativeUnusual() error = %v", err)
	}
	if result.Content != "options" || result.Method != "get_derivative_unusual" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGetUnusualRejectsEmptySymbolAndLongWindow(t *testing.T) {
	svc := NewSkillWrapService(client.New("https://example.invalid"))
	if _, err := svc.GetTechnicalUnusual(context.Background(), UnusualRequest{}); err == nil {
		t.Fatal("expected missing symbol error")
	}
	_, err := svc.GetFinancialUnusual(context.Background(), UnusualRequest{
		StockSymbol: "HK.00700",
		TimeRange:   366,
	})
	if err == nil {
		t.Fatal("expected time_range error")
	}
}

func TestGetDerivativeUnusualGatewayError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1.0/quote/HK.00700/unusual/derivatives" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"ret_code":-3,"ret_msg":"invalid request body","error":{"code":"invalid_parameter","message":"invalid request body"}}`)
	}))
	defer server.Close()

	svc := NewSkillWrapService(client.New(server.URL))
	_, err := svc.GetDerivativeUnusual(context.Background(), UnusualRequest{StockSymbol: "HK.00700"})
	if err == nil {
		t.Fatal("expected gateway error")
	}
}
