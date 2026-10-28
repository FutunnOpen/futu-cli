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
		if r.URL.Path != "/api/v1/get_technical_unusual" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["stock_symbol"] != "HK.00700" || body["time_range"] != float64(7) {
			t.Fatalf("body = %#v", body)
		}
		fmt.Fprint(w, `{"code":0,"message":"","time_range":"2026.1.1-2026.1.8","content":"ok"}`)
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
	if result.Content != "ok" || result.Method != "get_technical_unusual" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGetFinancialUnusualWrappedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"s":"ok","d":{"code":0,"message":"","time_range":"x","content":"wrapped"}}`)
	}))
	defer server.Close()

	svc := NewSkillWrapService(client.New(server.URL))
	result, err := svc.GetFinancialUnusual(context.Background(), UnusualRequest{StockSymbol: "US.AAPL"})
	if err != nil {
		t.Fatalf("GetFinancialUnusual() error = %v", err)
	}
	if result.Content != "wrapped" {
		t.Fatalf("content = %q", result.Content)
	}
}
