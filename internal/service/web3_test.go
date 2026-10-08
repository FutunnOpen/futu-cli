package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

const preciseWeb3Amount = "0.123456789012345678901234567890"

func TestWeb3GetBalanceEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/v1/web3/wallet/balance" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if got := request.URL.Query().Get("address"); got != "0x123" {
			t.Fatalf("address = %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"data":{"list":[{"token":"ETH","balance":1.5}]}}`)
	}))
	defer server.Close()

	balances, err := NewWeb3Service(client.New(server.URL)).GetBalance(context.Background(), "0x123")
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if len(balances) != 1 || balances[0].Token != "ETH" {
		t.Fatalf("balances = %#v", balances)
	}
}

func TestWeb3TradingEndpointsPreserveAmountString(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		invoke   func(*Web3Service) error
		validate func(*testing.T, map[string]any)
	}{
		{
			name: "quote", path: "/v1/web3/swap/quote",
			invoke: func(service *Web3Service) error {
				_, err := service.GetSwapQuote(context.Background(), "ETH", "USDC", preciseWeb3Amount)
				return err
			},
			validate: validateSwapBody,
		},
		{
			name: "execute", path: "/v1/web3/swap/execute",
			invoke: func(service *Web3Service) error {
				_, err := service.ExecuteSwap(context.Background(), "ETH", "USDC", preciseWeb3Amount)
				return err
			},
			validate: validateSwapBody,
		},
		{
			name: "transfer", path: "/v1/web3/transfer",
			invoke: func(service *Web3Service) error {
				_, err := service.Transfer(context.Background(), "0x456", "ETH", preciseWeb3Amount)
				return err
			},
			validate: func(t *testing.T, body map[string]any) {
				if body["to"] != "0x456" || body["token"] != "ETH" {
					t.Fatalf("transfer body = %#v", body)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != test.path {
					t.Fatalf("request = %s %s", request.Method, request.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if amount, ok := body["amount"].(string); !ok || amount != preciseWeb3Amount {
					t.Fatalf("amount = %#v, want exact JSON string", body["amount"])
				}
				test.validate(t, body)
				writer.Header().Set("Content-Type", "application/json")
				fmt.Fprint(writer, `{"code":0,"data":{}}`)
			}))
			defer server.Close()

			if err := test.invoke(NewWeb3Service(client.New(server.URL))); err != nil {
				t.Fatalf("invoke error = %v", err)
			}
		})
	}
}

func TestWeb3TradingReturnsHTTPAndAPIError(t *testing.T) {
	tests := []struct {
		name     string
		response string
		status   int
		want     string
	}{
		{name: "http", status: http.StatusBadGateway, response: `gateway unavailable`, want: "502"},
		{name: "api", status: http.StatusOK, response: `{"code":42,"message":"denied"}`, want: "denied"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				fmt.Fprint(writer, test.response)
			}))
			defer server.Close()

			_, err := NewWeb3Service(client.New(server.URL)).Transfer(
				context.Background(), "0x456", "ETH", preciseWeb3Amount,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func validateSwapBody(t *testing.T, body map[string]any) {
	t.Helper()
	if body["from_token"] != "ETH" || body["to_token"] != "USDC" {
		t.Fatalf("swap body = %#v", body)
	}
}
