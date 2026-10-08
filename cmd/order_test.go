package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/FutunnOpen/futu-cli/internal/service"
)

func TestSingleOrderColumnsOmitsEmptyValues(t *testing.T) {
	order := &service.Order{
		OrderID: "FH1D27CBC2D97F2000",
		Type:    "MODIFIED",
		Price:   301,
		Qty:     1,
	}

	headers, row := singleOrderColumns(order)

	wantHeaders := []string{orderHeaderID, orderHeaderType, orderHeaderPrice, orderHeaderQty}
	wantRow := []string{"FH1D27CBC2D97F2000", "MODIFIED(已改单)", "301.000", "1"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if !reflect.DeepEqual(row, wantRow) {
		t.Fatalf("row = %#v, want %#v", row, wantRow)
	}
}

func TestSingleOrderColumnsKeepsKnownOrderFields(t *testing.T) {
	order := &service.Order{
		OrderID: "FH1D27CBC2D97F2000",
		Symbol:  "US.AAPL",
		Side:    "BUY",
		Type:    "LIMIT",
		Price:   300,
		Qty:     1,
	}

	headers, row := singleOrderColumns(order)

	wantHeaders := []string{orderHeaderID, orderHeaderSymbol, orderHeaderSide, orderHeaderType, orderHeaderPrice, orderHeaderQty}
	wantRow := []string{"FH1D27CBC2D97F2000", "US.AAPL", "BUY(买入)", "LIMIT(限价单)", "300.000", "1"}
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("headers = %#v, want %#v", headers, wantHeaders)
	}
	if !reflect.DeepEqual(row, wantRow) {
		t.Fatalf("row = %#v, want %#v", row, wantRow)
	}
}

func TestOrderToRowFormatsMicrosecondTimestamp(t *testing.T) {
	const timestampSeconds = int64(1709251200)
	const timestampMicroseconds = int64(1709251200000000)

	order := &service.Order{
		OrderID:    "FH1D27CBC2D97F2000",
		Symbol:     "US.AAPL",
		Side:       "BUY",
		Type:       "LIMIT",
		Price:      300,
		Qty:        1,
		CreateTime: timestampMicroseconds,
	}

	row := orderToRow(order)

	if row[len(row)-1] != formatTimestamp(timestampSeconds) {
		t.Fatalf("formatted time = %q", row[len(row)-1])
	}
}

func TestBuildOrderDetailsRequest(t *testing.T) {
	oldFlags := orderFlags
	oldCfg := cfg
	defer func() {
		orderFlags = oldFlags
		cfg = oldCfg
	}()
	cfg = nil
	orderFlags.account = "123"
	orderFlags.exchange = "us"

	req, err := buildOrderDetailsRequest([]string{"ORD001", "ORD002"})
	if err != nil {
		t.Fatalf("buildOrderDetailsRequest() error = %v", err)
	}
	if req.Account != "123" || req.Exchange != "US" {
		t.Fatalf("request identity = %#v", req)
	}
	wantOrderIDs := []string{"ORD001", "ORD002"}
	if !reflect.DeepEqual(req.OrderIDs, wantOrderIDs) {
		t.Fatalf("order IDs = %#v, want %#v", req.OrderIDs, wantOrderIDs)
	}
}

func TestFormatDetailValueFormatsNestedValues(t *testing.T) {
	value := map[string]any{"name": "AAPL", "enabled": true}

	got := formatDetailValue("metadata", value)

	if !strings.Contains(got, `"name":"AAPL"`) || !strings.Contains(got, `"enabled":true`) {
		t.Fatalf("formatted detail value = %q", got)
	}
}

func TestFormatDetailValueFormatsExtraTimeFields(t *testing.T) {
	got := formatDetailValue("updated_time", float64(1709251200000000))

	if got != formatTimestamp(1709251200) {
		t.Fatalf("formatted time = %q", got)
	}
}

func TestBuildPlaceOrderRequestDefaultsToMarket(t *testing.T) {
	oldFlags, oldCfg := orderFlags, cfg
	defer func() { orderFlags, cfg = oldFlags, oldCfg }()
	cfg = nil
	orderFlags.price = ""
	orderFlags.auxPrice = ""
	orderFlags.typ = defaultOrderType
	orderFlags.account = "123"
	orderFlags.timeInForce = defaultTimeInForce
	orderFlags.lotType = ""
	orderFlags.remark = ""
	orderFlags.session = ""
	orderFlags.orderClass = ""
	orderFlags.multiLegInfo = ""

	req, err := buildPlaceOrderRequest("US.AAPL", "1", orderSideBuy)
	if err != nil {
		t.Fatalf("buildPlaceOrderRequest() error = %v", err)
	}
	if req.OrderType != orderTypeMarket || req.Price != "" || req.Code != "US.AAPL" {
		t.Fatalf("request = %#v", req)
	}
}

func TestBuildPlaceOrderRequestPreservesPriceAndAdvancedFields(t *testing.T) {
	oldFlags, oldCfg := orderFlags, cfg
	defer func() { orderFlags, cfg = oldFlags, oldCfg }()
	cfg = nil
	orderFlags.account = "123"
	orderFlags.typ = "LIMIT_IF_TOUCHED"
	orderFlags.price = "0.123456789012345678"
	orderFlags.auxPrice = "0.100000000000000001"
	orderFlags.timeInForce = "gtc"
	orderFlags.lotType = "odd"
	orderFlags.remark = "test"
	orderFlags.session = "OVERNIGHT"
	orderFlags.orderClass = ""
	orderFlags.multiLegInfo = `{"strategy":"custom"}`

	req, err := buildPlaceOrderRequest("US.AAPL", "2", orderSideSellShort)
	if err != nil {
		t.Fatalf("buildPlaceOrderRequest() error = %v", err)
	}
	if req.Price != orderFlags.price || req.AuxPrice != orderFlags.auxPrice || req.Side != orderSideSellShort || req.LotType != "ODD" {
		t.Fatalf("request = %#v", req)
	}
}

func TestNormalizeOrderTypeRejectsUnknownType(t *testing.T) {
	if _, err := normalizeOrderType("not-an-order"); err == nil {
		t.Fatal("normalizeOrderType() expected error")
	}
}

func TestConfirmActionAcceptsExplicitYes(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdin := os.Stdin
	defer func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	}()
	os.Stdin = reader
	if _, err := writer.WriteString("yes\n"); err != nil {
		t.Fatalf("write confirmation: %v", err)
	}
	_ = writer.Close()
	if !confirmAction("Confirm test?") {
		t.Fatal("confirmAction() rejected explicit yes")
	}
}
