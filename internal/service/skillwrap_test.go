package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/futunn/futu/internal/client"
)

func TestGetTechnicalUnusualUsesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1.0/quote/HK.00700/unusual/technical" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		var body unusualPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.TimeRange != 7 || body.LanguageID != 0 {
			t.Fatalf("body = %#v", body)
		}
		if len(body.IndicatorFilters) != 1 || body.IndicatorFilters[0] != "MACD" {
			t.Fatalf("indicator_filters = %#v", body.IndicatorFilters)
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
		if r.Method != http.MethodPost {
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

func TestGetDerivativeUnusualBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		var body unusualPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.AnalysisDimensions) != 2 || body.AnalysisDimensions[0] != "option_unusual" || body.AnalysisDimensions[1] != "option_volatility" {
			t.Fatalf("analysis_dimensions = %#v", body.AnalysisDimensions)
		}
		if body.TimeRange != 30 {
			t.Fatalf("time_range = %d", body.TimeRange)
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
		if r.Method != http.MethodPost {
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
