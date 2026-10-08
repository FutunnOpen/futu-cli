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

func TestGetCryptoTotalBalanceUsesAccountPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/total-balance" {
			t.Errorf("path = %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"cash_list":[{"currency":"USD","cash":"100.12"}],"positions":[{"coin":"BTC","qty":"0.5","market_val":"30000"}]}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	balance, err := cryptoService.GetTotalBalance(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetTotalBalance() error = %v", err)
	}
	if len(balance.CashList) != 1 || balance.CashList[0]["currency"] != "USD" {
		t.Fatalf("cash list = %#v", balance.CashList)
	}
	if len(balance.Positions) != 1 || balance.Positions[0]["coin"] != "BTC" {
		t.Fatalf("positions = %#v", balance.Positions)
	}
}

func TestGetCryptoTotalBalanceAcceptsAccountAPIShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"cash_list":[{"currency":"USD"}],"positions":[]}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	balance, err := cryptoService.GetTotalBalance(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetTotalBalance() error = %v", err)
	}
	if len(balance.CashList) != 1 {
		t.Fatalf("cash list = %#v", balance.CashList)
	}
}

func TestGetCryptoTotalBalanceReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.GetTotalBalance(context.Background(), "123")
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestGetCryptoMaxQtyUsesAccountPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/purchasing-power" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("order_type") != "LIMIT" {
			t.Errorf("order_type = %q", query.Get("order_type"))
		}
		if query.Get("currency") != "USD" {
			t.Errorf("currency = %q", query.Get("currency"))
		}
		if query.Get("coin") != "BTC" {
			t.Errorf("coin = %q", query.Get("coin"))
		}
		if query.Get("price") != "65000" {
			t.Errorf("price = %q", query.Get("price"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"max_cash_buy_qty":"0.12","max_sell_qty":"0.03","max_cash_buy_amount":"7800"}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	maxQty, err := cryptoService.GetMaxQty(context.Background(), CryptoMaxQtyRequest{
		Account: "123", OrderType: "LIMIT", Currency: "USD", Coin: "BTC", Price: "65000",
	})
	if err != nil {
		t.Fatalf("GetMaxQty() error = %v", err)
	}
	if maxQty.MaxCashBuyQty != "0.12" || maxQty.MaxSellQty != "0.03" {
		t.Fatalf("max qty = %#v", maxQty)
	}
}

func TestGetCryptoMaxQtyReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.GetMaxQty(context.Background(), CryptoMaxQtyRequest{
		Account: "123",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestListActiveCryptoOrdersUsesAccountPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("page_size") != "20" {
			t.Errorf("page_size = %q", query.Get("page_size"))
		}
		if query.Get("page_flag") != "next-1" {
			t.Errorf("page_flag = %q", query.Get("page_flag"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"orders":[{"order_id":"C001","symbol":"BTCUSD","side":"BUY","ord_type":"LIMIT","price":"60000","qty":"0.01","status":"SUBMITTED"}],"page_flag":"next-2","completed":false}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.ListActiveOrders(context.Background(), CryptoOrdersRequest{
		Account: "123", PageSize: 20, PageFlag: "next-1",
	})
	if err != nil {
		t.Fatalf("ListActiveOrders() error = %v", err)
	}
	if len(result.Orders) != 1 || result.Orders[0]["order_id"] != "C001" {
		t.Fatalf("orders = %#v", result.Orders)
	}
	if result.PageFlag != "next-2" || result.Completed {
		t.Fatalf("pagination = %#v", result)
	}
}

func TestListActiveCryptoOrdersReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.ListActiveOrders(context.Background(), CryptoOrdersRequest{
		Account: "123",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestListHistoryCryptoOrdersUsesAccountPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders/history" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		want := map[string]string{
			"start_time":   "1709251200000000",
			"end_time":     "1709337600000000",
			"page_size":    "20",
			"page_flag":    "next-1",
			"symbol":       "ETHHKD",
			"order_status": "FILLED_ALL,CANCELLED_ALL",
			"currency":     "HKD",
			"side":         "SELL",
			"ord_type":     "LIMIT",
		}
		for key, value := range want {
			if query.Get(key) != value {
				t.Errorf("%s = %q, want %q", key, query.Get(key), value)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"orders":[{"order_id":"C001"}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.ListHistoryOrders(context.Background(), CryptoOrdersRequest{
		Account: "123", Start: 1709251200000000, End: 1709337600000000,
		PageSize: 20, PageFlag: "next-1", Symbol: "ETHHKD", OrderStatus: "FILLED_ALL,CANCELLED_ALL",
		Currency: "HKD", Side: "SELL", OrderType: "LIMIT",
	})
	if err != nil {
		t.Fatalf("ListHistoryOrders() error = %v", err)
	}
	if len(result.Orders) != 1 || !result.Completed {
		t.Fatalf("result = %#v", result)
	}
}

func TestListHistoryCryptoOrdersReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.ListHistoryOrders(context.Background(), CryptoOrdersRequest{
		Account: "123",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestGetCryptoOrderDetailsUsesAccountPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders/C001,C002" {
			t.Errorf("path = %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"orders":[{"order_id":"C001"},{"order_id":"C002"}]}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	orders, err := cryptoService.GetOrderDetails(context.Background(), "123", []string{"C001", "C002"})
	if err != nil {
		t.Fatalf("GetOrderDetails() error = %v", err)
	}
	if len(orders) != 2 || orders[0]["order_id"] != "C001" || orders[1]["order_id"] != "C002" {
		t.Fatalf("orders = %#v", orders)
	}
}

func TestGetCryptoOrderDetailsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.GetOrderDetails(context.Background(), "123", []string{"C001"})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestListCryptoFillsUsesAccountPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/fills" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("order_id") != "C001" {
			t.Errorf("order_id = %q", query.Get("order_id"))
		}
		if query.Get("page_size") != "20" {
			t.Errorf("page_size = %q", query.Get("page_size"))
		}
		if query.Get("page_flag") != "next-1" {
			t.Errorf("page_flag = %q", query.Get("page_flag"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"order_fills":[{"fill_id":"F001","order_id":"C001"}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.ListFills(context.Background(), CryptoFillsRequest{
		Account: "123", OrderID: "C001", PageSize: 20, PageFlag: "next-1",
	})
	if err != nil {
		t.Fatalf("ListFills() error = %v", err)
	}
	if len(result.OrderFills) != 1 || result.OrderFills[0]["fill_id"] != "F001" {
		t.Fatalf("fills = %#v", result.OrderFills)
	}
}

func TestListCryptoFillsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.ListFills(context.Background(), CryptoFillsRequest{
		Account: "123", OrderID: "C001",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestListHistoryCryptoFillsAcceptsFillsField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/crypto/accounts/123/fills/history" {
			t.Errorf("path = %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"fills":[{"fill_id":"F001","order_id":"C001"}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.ListHistoryFills(context.Background(), CryptoFillsRequest{
		Account: "123", Start: 1743436800000000, End: 1751299199999999,
	})
	if err != nil {
		t.Fatalf("ListHistoryFills() error = %v", err)
	}
	if len(result.OrderFills) != 1 || result.OrderFills[0]["fill_id"] != "F001" {
		t.Fatalf("fills = %#v", result.OrderFills)
	}
}

func TestPlaceCryptoOrderUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		assertCryptoOrderPayload(t, body)
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"order_id":"C001"}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.PlaceOrder(context.Background(), CryptoPlaceOrderRequest{
		Account:         "123",
		ReqID:           "req-1234567890123",
		Side:            "BUY",
		Symbol:          "BTCUSD",
		OrderType:       "LIMIT",
		TimeInForce:     "TIF_GTC",
		Price:           "65000",
		Qty:             "0.01",
		ExpireTime:      "1785513600000000",
		ConditionalInfo: map[string]any{"trigger_price": "66000"},
	})
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}
	if result.OrderID != "C001" {
		t.Fatalf("order ID = %q", result.OrderID)
	}
}

func assertCryptoOrderPayload(t *testing.T, body map[string]any) {
	t.Helper()
	want := map[string]string{
		"req_id":        "req-1234567890123",
		"side":          "BUY",
		"symbol":        "BTCUSD",
		"ord_type":      "LIMIT",
		"time_in_force": "TIF_GTC",
		"price":         "65000",
		"qty":           "0.01",
		"expire_time":   "1785513600000000",
	}
	for key, value := range want {
		if body[key] != value {
			t.Fatalf("%s = %#v, want %q", key, body[key], value)
		}
	}
	conditional, ok := body["conditional_info"].(map[string]any)
	if !ok || conditional["trigger_price"] != "66000" {
		t.Fatalf("conditional_info = %#v", body["conditional_info"])
	}
}

func TestPlaceCryptoOrderOmitsOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		for _, key := range []string{"price", "qty", "cash_order_qty", "expire_time", "conditional_info"} {
			if _, ok := body[key]; ok {
				t.Fatalf("unexpected %s in body %#v", key, body)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_id":"C002"}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.PlaceOrder(context.Background(), CryptoPlaceOrderRequest{
		Account: "123", ReqID: "req-1234567890124", Side: "BUY", Symbol: "BTCUSD",
		OrderType: "MARKET", TimeInForce: "TIF_IOC",
	})
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}
	if result.OrderID != "C002" {
		t.Fatalf("order ID = %q", result.OrderID)
	}
}

func TestPlaceCryptoOrderReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.PlaceOrder(context.Background(), CryptoPlaceOrderRequest{
		Account: "123", ReqID: "req-1234567890125", Side: "BUY", Symbol: "BTCUSD",
		OrderType: "LIMIT", TimeInForce: "TIF_GTC",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestPlaceCryptoOrderReturnsMsgFieldError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":550000012,"msg":"insufficient buying power"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.PlaceOrder(context.Background(), CryptoPlaceOrderRequest{
		Account: "123", ReqID: "req-1234567890124", Side: "BUY", Symbol: "BTCUSD",
		OrderType: "LIMIT", TimeInForce: "TIF_GTC", Price: "60000", Qty: "0.01",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
	if !strings.Contains(err.Error(), "insufficient buying power") {
		t.Fatalf("error = %v", err)
	}
}

func TestModifyCryptoOrderUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders/C001" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		assertCryptoModifyOrderPayload(t, body)
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"order_id":"C001"}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.ModifyOrder(context.Background(), "C001", CryptoModifyOrderRequest{
		Account:         "123",
		ReqID:           "req-1234567890126",
		Price:           "65100",
		Qty:             "0.02",
		OrderVersion:    3,
		ConditionalInfo: map[string]any{"trigger_price": "66000"},
	})
	if err != nil {
		t.Fatalf("ModifyOrder() error = %v", err)
	}
	if result.OrderID != "C001" {
		t.Fatalf("order ID = %q", result.OrderID)
	}
}

func assertCryptoModifyOrderPayload(t *testing.T, body map[string]any) {
	t.Helper()
	want := map[string]string{
		"req_id": "req-1234567890126",
		"price":  "65100",
		"qty":    "0.02",
	}
	for key, value := range want {
		if body[key] != value {
			t.Fatalf("%s = %#v, want %q", key, body[key], value)
		}
	}
	if body["order_version"] != float64(3) {
		t.Fatalf("order_version = %#v, want 3", body["order_version"])
	}
	conditional, ok := body["conditional_info"].(map[string]any)
	if !ok || conditional["trigger_price"] != "66000" {
		t.Fatalf("conditional_info = %#v", body["conditional_info"])
	}
}

func TestModifyCryptoOrderReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":400,"errmsg":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.ModifyOrder(context.Background(), "C001", CryptoModifyOrderRequest{
		Account: "123", ReqID: "req-1234567890127", Price: "65100", Qty: "0.02",
		OrderVersion: 3, ConditionalInfo: map[string]any{"trigger_price": "66000"},
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestCancelCryptoOrderUsesAccountPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", request.Method)
		}
		if request.URL.Path != "/api/v1/crypto/accounts/123/orders/C001" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("req_id") != "req-1234567890128" {
			t.Fatalf("req_id = %#v", request.URL.Query().Get("req_id"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":0,"message":"","data":{"order_id":"C001"}}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	result, err := cryptoService.CancelOrder(context.Background(), "C001", CryptoCancelOrderRequest{
		Account: "123",
		ReqID:   "req-1234567890128",
	})
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
	if result.OrderID != "C001" {
		t.Fatalf("order ID = %q", result.OrderID)
	}
}

func TestCancelCryptoOrderReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"code":400,"message":"bad request"}`)
	}))
	defer server.Close()

	cryptoService := NewCryptoService(client.New(server.URL))
	_, err := cryptoService.CancelOrder(context.Background(), "C001", CryptoCancelOrderRequest{
		Account: "123",
		ReqID:   "req-1234567890129",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}
