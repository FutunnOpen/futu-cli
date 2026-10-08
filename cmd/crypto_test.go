package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/FutunnOpen/futu-cli/internal/service"
)

func TestCryptoMutationOrderIDFallsBackToArgument(t *testing.T) {
	got := cryptoMutationOrderID("FTHC3003082028640844800", &service.CryptoOrderResult{})

	if got != "FTHC3003082028640844800" {
		t.Fatalf("order ID = %q", got)
	}
}

func TestCryptoMutationOrderIDUsesResponseValue(t *testing.T) {
	got := cryptoMutationOrderID("fallback", &service.CryptoOrderResult{OrderID: "response-id"})

	if got != "response-id" {
		t.Fatalf("order ID = %q", got)
	}
}

func TestCryptoOrderSymbolBuildsFromCoinAndCurrency(t *testing.T) {
	order := service.CryptoOrder{
		"base_currency":  "ETH",
		"quote_currency": "HKD",
	}

	if got := cryptoOrderSymbol(order); got != "ETHHKD" {
		t.Fatalf("symbol = %q", got)
	}
}

func TestCryptoOrderSymbolReadsNestedSymbolPair(t *testing.T) {
	order := service.CryptoOrder{
		"symbol_pair": map[string]any{
			"base":   "ETH",
			"quote":  "HKD",
			"symbol": "ETHHKD",
		},
	}

	if got := cryptoOrderSymbol(order); got != "ETHHKD" {
		t.Fatalf("symbol = %q", got)
	}
}

func TestSortedCryptoOrderDetailKeysSkipsRenderedSymbolPair(t *testing.T) {
	order := service.CryptoOrder{
		"order_id":    "FTHC3003085223713821952",
		"symbol_pair": map[string]any{"symbol": "ETHHKD"},
		"price":       "50000",
	}

	keys := sortedCryptoOrderDetailKeys(order)

	if len(keys) != 1 || keys[0] != "price" {
		t.Fatalf("keys = %#v", keys)
	}
}

func TestFormatCryptoDetailValueSkipsEmptyAndZeroTriggerTime(t *testing.T) {
	if got := formatCryptoDetailValue("cash_order_qty", ""); got != "" {
		t.Fatalf("empty value = %q", got)
	}
	if got := formatCryptoDetailValue("trigger_time", float64(0)); got != "" {
		t.Fatalf("zero trigger time = %q", got)
	}
}

func TestFormatCryptoDetailValueFormatsTime(t *testing.T) {
	got := formatCryptoDetailValue("create_time", float64(1789978279892001))

	if got == "1789978279892001" || got == "" {
		t.Fatalf("formatted time = %q", got)
	}
}

func TestCryptoOrderFieldAcceptsAlternativeQuantityKeys(t *testing.T) {
	order := service.CryptoOrder{
		"order_quantity": float64(0.0212),
	}

	if got := cryptoOrderField(order, "qty", "order_quantity"); got != "0.0212" {
		t.Fatalf("quantity = %q", got)
	}
}

func TestFormatTimestampStringAcceptsScientificNotation(t *testing.T) {
	got := formatTimestampString("1.789978279892001e+15")

	if got == "1.789978279892001e+15" || got == "-" {
		t.Fatalf("formatted time = %q", got)
	}
}

func TestCryptoHistoryTimeRangeDefaultsToThreeMonths(t *testing.T) {
	oldFlags := cryptoOrderFlags
	defer func() {
		cryptoOrderFlags = oldFlags
	}()
	end := time.Date(2026, time.September, 21, 16, 30, 0, 0, time.Local).UnixMicro()
	wantStart := time.UnixMicro(end).AddDate(0, -defaultCryptoHistoryMonths, 0).UnixMicro()
	cryptoOrderFlags.start = 0
	cryptoOrderFlags.end = end

	start, gotEnd := cryptoHistoryTimeRange()

	if gotEnd != end {
		t.Fatalf("end = %d, want %d", gotEnd, end)
	}
	if start != wantStart {
		t.Fatalf("start = %d, want %d", start, wantStart)
	}
}

func TestRenderCryptoOrderListResultPrintsEmptyMessage(t *testing.T) {
	got := captureStdout(t, func() {
		renderCryptoOrderListResult(&service.CryptoOrderList{Completed: true, PageFlag: "0"})
	})

	if got != "No orders found.\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestRenderCryptoPageInfoHidesCompletedZeroCursor(t *testing.T) {
	got := captureStdout(t, func() {
		renderCryptoPageInfo(&service.CryptoOrderList{Completed: true, PageFlag: "0"})
	})

	if got != "" {
		t.Fatalf("output = %q", got)
	}
}

func TestCryptoFillSymbolReadsNestedSymbolPair(t *testing.T) {
	fill := service.CryptoFill{
		"symbol_pair": map[string]any{
			"symbol": "ETHHKD",
		},
	}

	if got := cryptoFillSymbol(fill); got != "ETHHKD" {
		t.Fatalf("symbol = %q", got)
	}
}

func TestFormatCryptoFillTimeReadsFillTime(t *testing.T) {
	fill := service.CryptoFill{"fill_time": float64(1747209825000000)}

	if got := formatCryptoFillTime(fill); got != "2025-05-14 16:03:45" {
		t.Fatalf("time = %q", got)
	}
}

func TestSummarizeCryptoFillsByOrder(t *testing.T) {
	fills := []service.CryptoFill{
		{
			"order_id":  "O1",
			"symbol":    "BTCUSD",
			"side":      "SELL",
			"fill_qty":  "0.0004",
			"amount":    "40",
			"fill_time": float64(1747209825000000),
		},
		{
			"order_id":  "O1",
			"symbol":    "BTCUSD",
			"side":      "SELL",
			"fill_qty":  "0.0006",
			"amount":    "60",
			"fill_time": float64(1747209830000000),
		},
	}

	summaries := summarizeCryptoFillsByOrder(fills)

	if len(summaries) != 1 {
		t.Fatalf("summary count = %d", len(summaries))
	}
	summary := summaries[0]
	if summary.OrderID != "O1" || summary.Symbol != "BTCUSD" || summary.Side != "SELL" {
		t.Fatalf("summary identity = %#v", summary)
	}
	if summary.Qty != 0.001 || summary.Amount != 100 {
		t.Fatalf("summary values = %#v", summary)
	}
	if got := formatCryptoAveragePrice(summary); got != "100000" {
		t.Fatalf("avg price = %q", got)
	}
}

func TestRenderCryptoFillListResultPrintsEmptyMessage(t *testing.T) {
	got := captureStdout(t, func() {
		renderCryptoFillListResult(&service.CryptoFillList{Completed: true, PageFlag: "0"})
	})

	if got != "No fills found.\n" {
		t.Fatalf("output = %q", got)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writeFile
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	if err := writeFile.Close(); err != nil {
		t.Fatalf("close stdout pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(readFile); err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return buf.String()
}
