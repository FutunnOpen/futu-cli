package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

func TestPlaceOrderUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("authorization header = %q", request.Header.Get("Authorization"))
		}

		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["code"] != "US.AAPL" || body["side"] != "BUY" || body["order_type"] != "LIMIT" {
			t.Fatalf("order identity body = %#v", body)
		}
		if body["qty"] != "100" || body["price"] != "150.25" || body["time_in_force"] != "DAY" {
			t.Fatalf("order value body = %#v", body)
		}

		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_id":"987654321"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL, client.WithToken("access-token")))
	req := PlaceOrderRequest{
		Account:     "123",
		Code:        "US.AAPL",
		Side:        "BUY",
		OrderType:   "LIMIT",
		Price:       "150.25",
		Qty:         100,
		TimeInForce: "DAY",
	}
	order, err := tradeService.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}
	if order.OrderID != "987654321" || order.Symbol != "US.AAPL" || order.Type != "LIMIT" {
		t.Fatalf("order = %#v", order)
	}
}

func TestPlaceOrderOmitsZeroPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["price"]; ok {
			t.Fatalf("unexpected price in body = %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_id":"987654321"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := PlaceOrderRequest{
		Account:     "123",
		Code:        "US.AAPL",
		Side:        "BUY",
		OrderType:   "MARKET",
		Qty:         100,
		TimeInForce: "DAY",
	}
	if _, err := tradeService.PlaceOrder(context.Background(), req); err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}
}

func TestPlaceOrderPayloadPreservesPrecisePricesAndAllFields(t *testing.T) {
	req := PlaceOrderRequest{
		Code: "US.AAPL", Side: "SELL_SHORT", OrderType: "LIMIT_IF_TOUCHED",
		Qty: 2, Price: "0.123456789012345678", AuxPrice: "0.100000000000000001",
		TimeInForce: "GTC", LotType: "ODD", Remark: "test", Session: "OVERNIGHT",
		OrderClass: "MLEG", MultiLegInfo: json.RawMessage(`{"strategy":"custom"}`),
	}
	payload := newPlaceOrderPayload(req)
	if payload.Price != req.Price || payload.AuxPrice != req.AuxPrice || payload.LotType != "ODD" || payload.Session != "OVERNIGHT" {
		t.Fatalf("payload = %#v", payload)
	}
	if string(payload.MultiLegInfo) != string(req.MultiLegInfo) {
		t.Fatalf("multi-leg info = %s", payload.MultiLegInfo)
	}
}

func TestModifyOrderPayloadPreservesPrecisePrices(t *testing.T) {
	req := ModifyOrderRequest{Exchange: "US", Qty: 1, Price: "0.123456789012345678", AuxPrice: "0.100000000000000001"}
	payload := newModifyOrderPayload(req)
	if payload.Price != req.Price || payload.AuxPrice != req.AuxPrice {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestPlaceOrderReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := PlaceOrderRequest{Account: "123", Code: "US.AAPL", Side: "BUY", OrderType: "LIMIT", Qty: 100, TimeInForce: "DAY"}
	_, err := tradeService.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestPlaceOrderReturnsConfirmationRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/api/v1.0/accounts/123/orders" {
			t.Fatalf("unexpected path = %s", request.URL.Path)
		}
		fmt.Fprint(writer, `{"s":"ok","d":{"need_order_confirm":true,"confirm_id":"confirm-1"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := PlaceOrderRequest{Account: "123", Code: "US.AAPL", Side: "BUY", OrderType: "LIMIT", Qty: 100, TimeInForce: "DAY"}
	_, err := tradeService.PlaceOrder(context.Background(), req)
	required, ok := err.(*OrderConfirmationRequiredError)
	if !ok {
		t.Fatalf("error = %T %v, want OrderConfirmationRequiredError", err, err)
	}
	if required.Account != "123" || required.ConfirmID != "confirm-1" || required.Exchange != "" {
		t.Fatalf("required confirmation = %#v", required)
	}
}

func TestModifyOrderUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders/987654321" {
			t.Errorf("path = %s", request.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["exchange"] != "US" || body["qty"] != "200" || body["price"] != "151.5" || body["aux_price"] != "149" {
			t.Fatalf("modify order body = %#v", body)
		}

		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_id":"987654321"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := ModifyOrderRequest{
		Account:  "123",
		Exchange: "US",
		Qty:      200,
		Price:    "151.5",
		AuxPrice: "149",
	}
	order, err := tradeService.ModifyOrder(context.Background(), "987654321", req)
	if err != nil {
		t.Fatalf("ModifyOrder() error = %v", err)
	}
	if order.OrderID != "987654321" || order.Qty != 200 || order.Price != 151.5 {
		t.Fatalf("order = %#v", order)
	}
}

func TestModifyOrderOmitsOptionalPrices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["price"]; ok {
			t.Fatalf("unexpected price in body = %#v", body)
		}
		if _, ok := body["aux_price"]; ok {
			t.Fatalf("unexpected aux_price in body = %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := ModifyOrderRequest{Account: "123", Exchange: "US", Qty: 200}
	order, err := tradeService.ModifyOrder(context.Background(), "987654321", req)
	if err != nil {
		t.Fatalf("ModifyOrder() error = %v", err)
	}
	if order.OrderID != "987654321" || order.Qty != 200 {
		t.Fatalf("order = %#v", order)
	}
}

func TestModifyOrderReturnsConfirmationRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/api/v1.0/accounts/123/orders/987654321" {
			t.Fatalf("unexpected path = %s", request.URL.Path)
		}
		fmt.Fprint(writer, `{"s":"ok","d":{"need_order_confirm":true,"confirm_id":"confirm-2"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := ModifyOrderRequest{Account: "123", Exchange: "US", Qty: 200, Price: "151.5"}
	_, err := tradeService.ModifyOrder(context.Background(), "987654321", req)
	required, ok := err.(*OrderConfirmationRequiredError)
	if !ok {
		t.Fatalf("error = %T %v, want OrderConfirmationRequiredError", err, err)
	}
	if required.Account != "123" || required.ConfirmID != "confirm-2" || required.Exchange != "US" {
		t.Fatalf("required confirmation = %#v", required)
	}
}

func TestCancelOrderUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders/987654321" {
			t.Errorf("path = %s", request.URL.Path)
		}

		if request.URL.Query().Get("exchange") != "US" {
			t.Fatalf("exchange query = %q", request.URL.Query().Get("exchange"))
		}

		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := CancelOrderRequest{Account: "123", Exchange: "US"}
	if err := tradeService.CancelOrder(context.Background(), "987654321", req); err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
}

func TestCancelOrderReturnsConfirmationRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/api/v1.0/accounts/123/orders/987654321" {
			t.Fatalf("unexpected path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("exchange") != "US" {
			t.Fatalf("exchange query = %q", request.URL.Query().Get("exchange"))
		}
		fmt.Fprint(writer, `{"s":"ok","d":{"need_order_confirm":true,"confirm_id":"confirm-3"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := CancelOrderRequest{Account: "123", Exchange: "US"}
	err := tradeService.CancelOrder(context.Background(), "987654321", req)
	required, ok := err.(*OrderConfirmationRequiredError)
	if !ok {
		t.Fatalf("error = %T %v, want OrderConfirmationRequiredError", err, err)
	}
	if required.Account != "123" || required.ConfirmID != "confirm-3" || required.Exchange != "US" {
		t.Fatalf("required confirmation = %#v", required)
	}
}

func TestConfirmOrderUsesConfirmEndpointAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/order_confirm" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode confirm body: %v", err)
		}
		if body["confirm_id"] != "confirm-1" || body["exchange"] != "US" {
			t.Fatalf("confirm body = %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_id":"987654321"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	orderID, err := tradeService.ConfirmOrder(context.Background(), OrderConfirmRequest{
		Account:   "123",
		ConfirmID: "confirm-1",
		Exchange:  "US",
	})
	if err != nil {
		t.Fatalf("ConfirmOrder() error = %v", err)
	}
	if orderID != "987654321" {
		t.Fatalf("order id = %q", orderID)
	}
}

func TestOrderUnmarshalAcceptsOfficialTradingFields(t *testing.T) {
	data := []byte(`{
		"order_id":"FH1D27CBC2D97F2000",
		"code":"US.AAPL",
		"trd_side":"BUY",
		"order_type":"LIMIT",
		"order_status":"SUBMITTED",
		"qty":"1",
		"price":"301",
		"dealt_qty":"0",
		"create_time":1789722000
	}`)

	var order Order
	if err := json.Unmarshal(data, &order); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if order.Symbol != "US.AAPL" || order.Side != "BUY" || order.Type != "LIMIT" || order.Status != "SUBMITTED" {
		t.Fatalf("order identity fields = %#v", order)
	}
	if order.Qty != 1 || order.Price != 301 || order.FilledQty != 0 {
		t.Fatalf("order value fields = %#v", order)
	}
}

func TestOrderUnmarshalPreservesExtraFields(t *testing.T) {
	data := []byte(`{
		"order_id":"FH1D27CBC2D97F2000",
		"code":"US.AAPL",
		"qty":"1",
		"remark":"detail",
		"trail_value":"0.5"
	}`)

	var order Order
	if err := json.Unmarshal(data, &order); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if order.Extra["remark"] != "detail" || order.Extra["trail_value"] != "0.5" {
		t.Fatalf("extra fields = %#v", order.Extra)
	}

	encoded, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var values map[string]any
	if err := json.Unmarshal(encoded, &values); err != nil {
		t.Fatalf("decode marshaled order: %v", err)
	}
	if values["remark"] != "detail" || values["trail_value"] != "0.5" {
		t.Fatalf("marshaled fields = %#v", values)
	}
}

func TestListAccountsUsesAuthorizedTradingAccountsAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/authorized_trd_accs" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("authorization header = %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"accounts":[{"account_id":1234567890123456,"security_firm":"FUTUINC","enable_market":[1,2],"univs_account_card_number":456,"acc_type":"margin","account_card_number":"789"}]}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL, client.WithToken("access-token")))
	accounts, err := tradeService.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts() error = %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %#v", accounts)
	}
	account := accounts[0]
	if account.AccountID != "1234567890123456" || account.AccountNo != "789" || account.Type != "margin" {
		t.Fatalf("account = %#v", account)
	}
	if account.UniversalAccountNo != "456" {
		t.Fatalf("universal account number = %q", account.UniversalAccountNo)
	}
	if account.SecurityFirm != "FUTUINC" || len(account.EnabledTradingMarkets) != 2 {
		t.Fatalf("account metadata = %#v", account)
	}
}

func TestListAccountsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.ListAccounts(context.Background())
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestGetPositionsUsesAccountPathAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/positions" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("code") != "US.AAPL" {
			t.Errorf("code = %q", request.URL.Query().Get("code"))
		}
		if request.URL.Query().Get("pl_ratio_min") != "-0.1" {
			t.Errorf("pl_ratio_min = %q", request.URL.Query().Get("pl_ratio_min"))
		}
		if request.URL.Query().Get("pl_ratio_max") != "0.2" {
			t.Errorf("pl_ratio_max = %q", request.URL.Query().Get("pl_ratio_max"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"positions":[{"position_side":"LONG","code":"US.AAPL","stock_name":"Apple Inc","qty":"100","can_sell_qty":"100","currency":"USD","nominal_price":"150.00","cost_price":"140.00","cost_price_valid":true,"market_val":"15000.00","pl_ratio":"0.0714","pl_ratio_valid":true,"pl_val":"1000.00","pl_val_valid":true,"today_pl_val":"200.00","today_trd_val":"0","today_buy_qty":"0","today_buy_val":"0","today_sell_qty":"0","today_sell_val":"0","unrealized_pl":"1000.00","realized_pl":"0"}]}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL, client.WithToken("access-token")))
	filter := PositionFilter{Code: "US.AAPL", PLRatioMin: "-0.1", PLRatioMax: "0.2"}
	positions, err := tradeService.GetPositions(context.Background(), "123", filter)
	if err != nil {
		t.Fatalf("GetPositions() error = %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("positions = %#v", positions)
	}
	p := positions[0]
	if p.Code != "US.AAPL" || p.StockName != "Apple Inc" || p.PositionSide != "LONG" {
		t.Fatalf("position identity = %#v", p)
	}
	if p.Qty != "100" || p.CostPrice != "140.00" || p.MarketVal != "15000.00" {
		t.Fatalf("position values = %#v", p)
	}
	if !p.PLValValid || p.PLVal != "1000.00" || !p.PLRatioValid || p.PLRatio != "0.0714" {
		t.Fatalf("position P/L = %#v", p)
	}
}

func TestGetPositionsAcceptsLegacyArrayData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":[{"position_side":"SHORT","code":"US.TSLA","stock_name":"Tesla","qty":"2"}]}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	positions, err := tradeService.GetPositions(context.Background(), "123", PositionFilter{})
	if err != nil {
		t.Fatalf("GetPositions() error = %v", err)
	}
	if len(positions) != 1 || positions[0].Code != "US.TSLA" || positions[0].PositionSide != "SHORT" {
		t.Fatalf("positions = %#v", positions)
	}
}

func TestGetPositionsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.GetPositions(context.Background(), "123", PositionFilter{})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestGetPositionsOmitsEmptyFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("code") != "" {
			t.Errorf("unexpected code param = %q", request.URL.Query().Get("code"))
		}
		if request.URL.Query().Get("pl_ratio_min") != "" {
			t.Errorf("unexpected pl_ratio_min param = %q", request.URL.Query().Get("pl_ratio_min"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"positions":[]}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	positions, err := tradeService.GetPositions(context.Background(), "456", PositionFilter{})
	if err != nil {
		t.Fatalf("GetPositions() error = %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("expected empty positions, got %d", len(positions))
	}
}

func TestListTodayDealsUsesAccountPathAndMarket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/order_fills" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("trd_market") != "US" {
			t.Fatalf("trd_market = %q", query.Get("trd_market"))
		}
		if _, ok := query["page_flag"]; ok {
			t.Fatalf("page_flag should be omitted for first page, query = %s", query.Encode())
		}
		if query.Get("page_size") != "20" {
			t.Fatalf("page_size = %q", query.Get("page_size"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D001","order_id":"ORD001","code":"US.AAPL","stock_name":"Apple","trd_side":"BUY","price":"180.000","qty":"100","create_time":1709251200000000,"updated_time":1709251300000000,"counter_broker_id":"12","counter_broker_name":"Broker","status":"OK"}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := TodayDealsRequest{Account: "123", Market: "US", PageSize: 20}
	deals, err := tradeService.ListTodayDeals(context.Background(), req)
	if err != nil {
		t.Fatalf("ListTodayDeals() error = %v", err)
	}
	if len(deals) != 1 {
		t.Fatalf("deals count = %d, want 1", len(deals))
	}
	deal := deals[0]
	if deal.DealID != "D001" || deal.OrderID != "ORD001" || deal.Symbol != "US.AAPL" {
		t.Fatalf("deal identity = %#v", deal)
	}
	if deal.Side != "BUY" || deal.Price != 180 || deal.Qty != 100 || deal.CounterBrokerID != 12 {
		t.Fatalf("deal values = %#v", deal)
	}
}

func TestListTodayDealsPaginatesUntilCompleted(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callCount++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_flag") {
		case "":
			fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D001"}],"page_flag":"page2","completed":false}}`)
		case "page2":
			fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D002"}],"page_flag":"","completed":true}}`)
		default:
			t.Fatalf("unexpected page_flag = %q", request.URL.Query().Get("page_flag"))
		}
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	deals, err := tradeService.ListTodayDeals(context.Background(), TodayDealsRequest{Account: "123"})
	if err != nil {
		t.Fatalf("ListTodayDeals() error = %v", err)
	}
	if len(deals) != 2 || deals[0].DealID != "D001" || deals[1].DealID != "D002" {
		t.Fatalf("deals = %#v", deals)
	}
	if callCount != 2 {
		t.Fatalf("call count = %d, want 2", callCount)
	}
}

func TestListTodayDealsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.ListTodayDeals(context.Background(), TodayDealsRequest{Account: "123"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestListHistoryDealsUsesAccountPathAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/fills_history" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("trd_market") != "US" || query.Get("code") != "US.AAPL" {
			t.Fatalf("identity query = %s", query.Encode())
		}
		if query.Get("start") != "1709251200000000" || query.Get("end") != "1709337600000000" {
			t.Fatalf("time query = %s", query.Encode())
		}
		if _, ok := query["page_flag"]; ok {
			t.Fatalf("page_flag should be omitted for first page, query = %s", query.Encode())
		}
		if query.Get("page_size") != "20" {
			t.Fatalf("page_size = %q", query.Get("page_size"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D001","order_id":"ORD001","code":"US.AAPL","stock_name":"Apple","trd_side":"BUY","price":"180.000","qty":"100","create_time":1709251200000000,"updated_time":1709251300000000,"status":"OK"}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := HistoryDealsRequest{
		Account:  "123",
		Market:   "US",
		Start:    1709251200000000,
		End:      1709337600000000,
		Symbol:   "US.AAPL",
		PageSize: 20,
	}
	deals, err := tradeService.ListHistoryDeals(context.Background(), req)
	if err != nil {
		t.Fatalf("ListHistoryDeals() error = %v", err)
	}
	if len(deals) != 1 || deals[0].DealID != "D001" || deals[0].Symbol != "US.AAPL" {
		t.Fatalf("deals = %#v", deals)
	}
}

func TestListHistoryDealsPaginatesUntilCompleted(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callCount++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_flag") {
		case "":
			fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D001"}],"page_flag":"page2","completed":false}}`)
		case "page2":
			fmt.Fprint(writer, `{"s":"ok","d":{"order_fills":[{"deal_id":"D002"}],"page_flag":"","completed":true}}`)
		default:
			t.Fatalf("unexpected page_flag = %q", request.URL.Query().Get("page_flag"))
		}
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	deals, err := tradeService.ListHistoryDeals(context.Background(), HistoryDealsRequest{Account: "123"})
	if err != nil {
		t.Fatalf("ListHistoryDeals() error = %v", err)
	}
	if len(deals) != 2 || deals[0].DealID != "D001" || deals[1].DealID != "D002" {
		t.Fatalf("deals = %#v", deals)
	}
	if callCount != 2 {
		t.Fatalf("call count = %d, want 2", callCount)
	}
}

func TestListHistoryDealsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.ListHistoryDeals(context.Background(), HistoryDealsRequest{Account: "123"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestGetFundsUsesAccountPathAndCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/funds" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("currency") != "HKD" {
			t.Errorf("currency = %q", request.URL.Query().Get("currency"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"power":"12500.00","total_assets":25000.5,"cash":"7000.00","market_val":"18000.00","currency":"HKD","available_funds":"7000.00","risk_status":"LEVEL2"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	funds, err := tradeService.GetFunds(context.Background(), "123", "HKD")
	if err != nil {
		t.Fatalf("GetFunds() error = %v", err)
	}
	if funds.Currency != "HKD" || funds.TotalAssets != "25000.5" || funds.Power != "12500.00" {
		t.Fatalf("funds = %#v", funds)
	}
}

func TestGetFundsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.GetFunds(context.Background(), "123", "USD")
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestListOpenOrdersUsesAccountPathAndMarket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("trd_market") != "US" {
			t.Fatalf("trd_market = %q", query.Get("trd_market"))
		}
		if _, ok := query["page_flag"]; ok {
			t.Fatalf("page_flag should be omitted for first page, query = %s", query.Encode())
		}
		if query.Get("page_size") != "20" {
			t.Fatalf("page_size = %q", query.Get("page_size"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD001","code":"US.AAPL","side":"BUY","order_type":"LIMIT","order_status":"SUBMITTED","qty":"100","price":"180.000","dealt_qty":"0","create_time":1709251200000000}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OpenOrdersRequest{Account: "123", Market: "US", PageSize: 20}
	orders, err := tradeService.ListOpenOrders(context.Background(), req)
	if err != nil {
		t.Fatalf("ListOpenOrders() error = %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders))
	}
	if orders[0].OrderID != "ORD001" || orders[0].Symbol != "US.AAPL" || orders[0].Side != "BUY" {
		t.Fatalf("order = %#v", orders[0])
	}
}

func TestListOpenOrdersDefaultsMarket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("trd_market") != DefaultTradingMarket {
			t.Fatalf("trd_market = %q", request.URL.Query().Get("trd_market"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"orders":[],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OpenOrdersRequest{Account: "123"}
	if _, err := tradeService.ListOpenOrders(context.Background(), req); err != nil {
		t.Fatalf("ListOpenOrders() error = %v", err)
	}
}

func TestListOpenOrdersPaginatesUntilCompleted(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callCount++
		query := request.URL.Query()
		writer.Header().Set("Content-Type", "application/json")
		switch query.Get("page_flag") {
		case "":
			fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD001","code":"US.AAPL","order_type":"LIMIT","order_status":"SUBMITTED","qty":"1","price":"100"}],"page_flag":"page2","completed":false}}`)
		case "page2":
			fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD002","code":"US.TSLA","order_type":"MARKET","order_status":"FILLED_ALL","qty":"5","price":"200"}],"page_flag":"","completed":true}}`)
		default:
			t.Fatalf("unexpected page_flag = %q", query.Get("page_flag"))
		}
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OpenOrdersRequest{Account: "456", Market: "HK"}
	orders, err := tradeService.ListOpenOrders(context.Background(), req)
	if err != nil {
		t.Fatalf("ListOpenOrders() error = %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders count = %d, want 2", len(orders))
	}
	if orders[0].OrderID != "ORD001" || orders[1].OrderID != "ORD002" {
		t.Fatalf("orders = %#v", orders)
	}
	if callCount != 2 {
		t.Fatalf("call count = %d, want 2", callCount)
	}
}

func TestListOpenOrdersReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.ListOpenOrders(context.Background(), OpenOrdersRequest{Account: "123", Market: "HK"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestListOpenOrdersOmitsInvalidPageSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("page_size") != "" {
			t.Fatalf("unexpected page_size = %q for out-of-range value", request.URL.Query().Get("page_size"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"orders":[],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OpenOrdersRequest{Account: "123", Market: "HK", PageSize: 5}
	orders, err := tradeService.ListOpenOrders(context.Background(), req)
	if err != nil {
		t.Fatalf("ListOpenOrders() error = %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected empty orders, got %d", len(orders))
	}
}

func TestListHistoryOrdersUsesAccountPathAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders_history" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("trd_market") != "US" || query.Get("code") != "US.AAPL" {
			t.Fatalf("identity query = %s", query.Encode())
		}
		if query.Get("start") != "1709251200000000" || query.Get("end") != "1709337600000000" {
			t.Fatalf("time query = %s", query.Encode())
		}
		if _, ok := query["page_flag"]; ok {
			t.Fatalf("page_flag should be omitted for first page, query = %s", query.Encode())
		}
		if query.Get("page_size") != "20" {
			t.Fatalf("page_size = %q", query.Get("page_size"))
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD001","code":"US.AAPL","trd_side":"BUY","order_type":"LIMIT","order_status":"FILLED_ALL","qty":"100","price":"180.000","dealt_qty":"100","create_time":1709251200000000}],"page_flag":"","completed":true}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := HistoryOrdersRequest{
		Account:  "123",
		Market:   "US",
		Start:    1709251200000000,
		End:      1709337600000000,
		Symbol:   "US.AAPL",
		PageSize: 20,
	}
	orders, err := tradeService.ListHistoryOrders(context.Background(), req)
	if err != nil {
		t.Fatalf("ListHistoryOrders() error = %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders))
	}
	if orders[0].OrderID != "ORD001" || orders[0].FilledQty != 100 || orders[0].Status != "FILLED_ALL" {
		t.Fatalf("order = %#v", orders[0])
	}
}

func TestListHistoryOrdersPaginatesUntilCompleted(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		callCount++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_flag") {
		case "":
			fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD001","code":"HK.00700"}],"page_flag":"page2","completed":false}}`)
		case "page2":
			fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD002","code":"HK.09988"}],"page_flag":"","completed":true}}`)
		default:
			t.Fatalf("unexpected page_flag = %q", request.URL.Query().Get("page_flag"))
		}
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := HistoryOrdersRequest{Account: "456", Market: "HK"}
	orders, err := tradeService.ListHistoryOrders(context.Background(), req)
	if err != nil {
		t.Fatalf("ListHistoryOrders() error = %v", err)
	}
	if len(orders) != 2 || orders[0].OrderID != "ORD001" || orders[1].OrderID != "ORD002" {
		t.Fatalf("orders = %#v", orders)
	}
	if callCount != 2 {
		t.Fatalf("call count = %d, want 2", callCount)
	}
}

func TestListHistoryOrdersReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.ListHistoryOrders(context.Background(), HistoryOrdersRequest{Account: "123", Market: "HK"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestGetOrderDetailsUsesAccountPathAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/orders/detail" {
			t.Errorf("path = %s", request.URL.Path)
		}
		var body struct {
			OrderIDs []string `json:"order_ids"`
			Exchange string   `json:"exchange"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Exchange != "US" {
			t.Fatalf("exchange = %q", body.Exchange)
		}
		if len(body.OrderIDs) != 2 || body.OrderIDs[0] != "ORD001" || body.OrderIDs[1] != "ORD002" {
			t.Fatalf("order_ids = %#v", body.OrderIDs)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"orders":[{"order_id":"ORD001","code":"US.AAPL","trd_side":"BUY","order_type":"LIMIT","order_status":"FILLED_ALL","qty":"100","price":"180.000","dealt_qty":"100","create_time":1709251200000000}]}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OrderDetailsRequest{Account: "123", Exchange: "us", OrderIDs: []string{"ORD001", "ORD002"}}
	orders, err := tradeService.GetOrderDetails(context.Background(), req)
	if err != nil {
		t.Fatalf("GetOrderDetails() error = %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders))
	}
	if orders[0].OrderID != "ORD001" || orders[0].Symbol != "US.AAPL" || orders[0].FilledQty != 100 {
		t.Fatalf("order = %#v", orders[0])
	}
}

func TestGetOrderDetailsAcceptsArrayData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":[{"order_id":"ORD001","code":"US.AAPL","trd_side":"BUY","order_type":"LIMIT","order_status":"FILLED_ALL","qty":"100","price":"180.000","dealt_qty":"100"}]}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := OrderDetailsRequest{Account: "123", Exchange: "US", OrderIDs: []string{"ORD001"}}
	orders, err := tradeService.GetOrderDetails(context.Background(), req)
	if err != nil {
		t.Fatalf("GetOrderDetails() error = %v", err)
	}
	if len(orders) != 1 || orders[0].OrderID != "ORD001" {
		t.Fatalf("orders = %#v", orders)
	}
}

func TestGetOrderDetailsReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.GetOrderDetails(context.Background(), OrderDetailsRequest{Account: "123"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}

func TestGetTradingInfoUsesAccountPathAndParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/api/v1.0/accounts/123/acctradinginfo" {
			t.Errorf("path = %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("code") != "US.AAPL" || query.Get("order_type") != "LIMIT" {
			t.Fatalf("identity query = %s", query.Encode())
		}
		if query.Get("price") != "150.250000000000000001" || query.Get("order_id") != "987654321" {
			t.Fatalf("value query = %s", query.Encode())
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{"max_cash_buy":"10","max_cash_and_margin_buy":"20","max_position_sell":"5","max_sell_short":"3","max_buy_back":"2","long_required_im":"100.00","short_required_im":"120.00"}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := TradingInfoRequest{
		Account:   "123",
		Code:      "US.AAPL",
		OrderType: "LIMIT",
		Price:     "150.250000000000000001",
		OrderID:   "987654321",
	}
	info, err := tradeService.GetTradingInfo(context.Background(), req)
	if err != nil {
		t.Fatalf("GetTradingInfo() error = %v", err)
	}
	if info.MaxCashBuy != "10" || info.MaxCashAndMarginBuy != "20" || info.MaxPositionSell != "5" {
		t.Fatalf("trading info quantities = %#v", info)
	}
	if info.LongRequiredIM != "100.00" || info.ShortRequiredIM != "120.00" {
		t.Fatalf("trading info margins = %#v", info)
	}
}

func TestGetTradingInfoOmitsOptionalParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("price") != "" || query.Get("order_id") != "" {
			t.Fatalf("unexpected optional query = %s", query.Encode())
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"ok","d":{}}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	req := TradingInfoRequest{Account: "123", Code: "US.AAPL", OrderType: "MARKET"}
	if _, err := tradeService.GetTradingInfo(context.Background(), req); err != nil {
		t.Fatalf("GetTradingInfo() error = %v", err)
	}
}

func TestGetTradingInfoReturnsAPIBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"s":"error","errcode":-1200,"errmsg":"permission denied"}`)
	}))
	defer server.Close()

	tradeService := NewTradeService(client.New(server.URL))
	_, err := tradeService.GetTradingInfo(context.Background(), TradingInfoRequest{Account: "123"})
	if err == nil {
		t.Fatal("expected account API error")
	}
}
