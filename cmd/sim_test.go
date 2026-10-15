package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/futunn/futu/internal/service"
)

func TestSimCommandRegistration(t *testing.T) {
	for _, args := range [][]string{
		{"sim", "account", "list"},
		{"sim", "account", "funds"},
		{"sim", "position"},
		{"sim", "order", "buy"},
		{"sim", "order", "sell"},
		{"sim", "order", "place"},
		{"sim", "order", "list"},
		{"sim", "order", "history"},
		{"sim", "order", "modify"},
		{"sim", "order", "cancel"},
		{"sim", "order", "max-qty"},
	} {
		if _, _, err := rootCmd.Find(args); err != nil {
			t.Fatalf("command %v not registered: %v", args, err)
		}
	}
}

func TestFlatSimTradeCommandsAreNotRegistered(t *testing.T) {
	for _, name := range []string{
		"sim-trade-account-list",
		"sim-trade-cash-info",
		"sim-trade-order-list",
		"sim-trade-history-order-list",
		"sim-trade-max-buy-sell",
		"sim-trade-input-order",
		"sim-trade-modify-order",
		"sim-trade-cancel-order",
		"sim-trade-position-list",
	} {
		if _, _, err := rootCmd.Find([]string{name}); err == nil {
			t.Fatalf("flat simulated-trading command %q is still registered", name)
		}
	}
}

func TestSimMarketValue(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{value: "", want: simMarketHongKong},
		{value: "HK", want: simMarketHongKong},
		{value: "us", want: simMarketUnitedStates},
		{value: "US_OPTION", want: simMarketUSOption},
		{value: "hkcc", want: simMarketHongKongConnect},
	}

	for _, test := range tests {
		got, err := simMarketValue(test.value)
		if err != nil {
			t.Fatalf("simMarketValue(%q) error = %v", test.value, err)
		}
		if got != test.want {
			t.Fatalf("simMarketValue(%q) = %d, want %d", test.value, got, test.want)
		}
	}
}

func TestSimOrderTypeAndSideValues(t *testing.T) {
	orderType, err := simOrderTypeValue("limit")
	if err != nil {
		t.Fatalf("simOrderTypeValue() error = %v", err)
	}
	if orderType != simOrderTypeLimit {
		t.Fatalf("order type = %d, want %d", orderType, simOrderTypeLimit)
	}

	side, err := simOrderSideValue("sell")
	if err != nil {
		t.Fatalf("simOrderSideValue() error = %v", err)
	}
	if side != simOrderSideSellValue {
		t.Fatalf("order side = %d, want %d", side, simOrderSideSellValue)
	}
}

func TestBuildSimPlaceOrderBody(t *testing.T) {
	oldFlags := simFlags
	defer func() { simFlags = oldFlags }()
	simFlags.market = "US"
	simFlags.typ = "limit"
	simFlags.price = "180.5"
	simFlags.text = "paper test"
	simFlags.session = "RTH"

	body, err := buildSimPlaceOrderBody("US.AAPL", "2", orderSideBuy)
	if err != nil {
		t.Fatalf("buildSimPlaceOrderBody() error = %v", err)
	}
	if body[simFieldMarket] != simMarketUnitedStates || body[simFieldSymbol] != "US.AAPL" {
		t.Fatalf("market/symbol body = %#v", body)
	}
	if body[simFieldOrderType] != simOrderTypeLimit || body[simFieldOrderSide] != simOrderSideBuyValue {
		t.Fatalf("order enum body = %#v", body)
	}
	if body[simFieldQuantity] != "2" || body[simFieldPrice] != "180.5" || body[simFieldText] != "paper test" {
		t.Fatalf("order value body = %#v", body)
	}
	if body[simFieldTradeTimeType] != simTradeTimeTypeRegular {
		t.Fatalf("session body = %#v", body)
	}
}

func TestBuildSimPlaceOrderBodyRejectsInvalidQuantity(t *testing.T) {
	oldFlags := simFlags
	defer func() { simFlags = oldFlags }()
	simFlags.market = defaultSimMarket
	simFlags.typ = defaultSimOrderType

	if _, err := buildSimPlaceOrderBody("US.AAPL", "0", orderSideBuy); err == nil {
		t.Fatal("expected invalid quantity error")
	}
}

func TestApplySimModifyBodyRequiresQuantityAndPrice(t *testing.T) {
	oldFlags := simFlags
	defer func() { simFlags = oldFlags }()

	tests := []struct {
		name       string
		quantity   string
		price      string
		wantErrMsg string
	}{
		{name: "missing quantity", price: "181", wantErrMsg: errSimModifyQuantityReq},
		{name: "missing price", quantity: "2", wantErrMsg: errSimModifyPriceReq},
		{name: "quantity and price", quantity: "2", price: "181"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			simFlags = oldFlags
			simFlags.quantity = test.quantity
			simFlags.price = test.price
			body := map[string]any{}

			err := applySimModifyBody(body)
			if test.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("applySimModifyBody() error = %v, want %q", err, test.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("applySimModifyBody() error = %v", err)
			}
			if body[simFieldNewQuantity] != test.quantity || body[simFieldNewPrice] != test.price {
				t.Fatalf("modify body = %#v", body)
			}
		})
	}
}

func TestParsePositiveSimQuantityRejectsNaN(t *testing.T) {
	if _, err := parsePositiveSimQuantity("NaN"); err == nil {
		t.Fatal("expected invalid NaN quantity error")
	}
}

func TestValidateSimOrderResultRejectsZeroOrderID(t *testing.T) {
	result := &service.BFFResult{Data: json.RawMessage(`{"order_id":"0"}`)}
	if _, err := validateSimOrderResult(simEndpointInputOrder, result); err == nil {
		t.Fatal("expected invalid order_id error")
	}
}

func TestValidateSimOrderResultAcceptsNonZeroOrderID(t *testing.T) {
	result := &service.BFFResult{Data: json.RawMessage(`{"order_id":"12345"}`)}
	got, err := validateSimOrderResult(simEndpointInputOrder, result)
	if err != nil {
		t.Fatalf("validateSimOrderResult() error = %v", err)
	}
	if got != "12345" {
		t.Fatalf("order_id = %q, want 12345", got)
	}
}

func TestSimOrderListContainsOrderID(t *testing.T) {
	result := &service.BFFResult{Data: json.RawMessage(`{"orders":[{"order_id":"12345"}]}`)}
	if !simOrderListContains(result, "12345") {
		t.Fatal("expected order list to contain order_id")
	}
	if simOrderListContains(result, "99999") {
		t.Fatal("unexpected order_id match")
	}
}

func TestSimOrderListContainsTopLevelOrderArray(t *testing.T) {
	result := &service.BFFResult{Data: json.RawMessage(`[{"order_id":"12345"}]`)}
	if !simOrderListContains(result, "12345") {
		t.Fatal("expected top-level order array to contain order_id")
	}
}
