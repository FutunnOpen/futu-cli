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

func TestGetTechnicalUnusualPostsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/v1.0/quote/HK.00700/unusual/technical" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["stock_symbol"]; ok {
			t.Fatalf("symbol must be a path parameter, body = %#v", body)
		}
		if body["time_range"] != float64(7) {
			t.Fatalf("body = %#v", body)
		}
		filters, _ := body["indicator_filters"].([]any)
		if len(filters) != 1 || filters[0] != "MACD" {
			t.Fatalf("indicator_filters = %#v", body["indicator_filters"])
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

func TestGetDerivativeUnusualGatewayError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
