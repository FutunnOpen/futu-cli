package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/futunn/futu/internal/client"
	"github.com/futunn/futu/internal/service"
)

func TestFormatSecurityFirm(t *testing.T) {
	if got := formatSecurityFirm(securityFirmFutuInc); got != "moomoo (United States)" {
		t.Fatalf("formatSecurityFirm() = %q", got)
	}
	if got := formatSecurityFirm("UNKNOWN"); got != "UNKNOWN" {
		t.Fatalf("unknown security firm = %q", got)
	}
}

func TestFormatTradingMarkets(t *testing.T) {
	markets := []int{marketHongKong, marketUnitedStates, 999}
	want := "Hong Kong, United States, 999"
	if got := formatTradingMarkets(markets); got != want {
		t.Fatalf("formatTradingMarkets() = %q, want %q", got, want)
	}
}

func TestFormatAccountType(t *testing.T) {
	if got := formatAccountType(accountTypeMargin); got != "Margin" {
		t.Fatalf("formatAccountType() = %q", got)
	}
	if got := formatAccountType("unknown"); got != "unknown" {
		t.Fatalf("unknown account type = %q", got)
	}
}

func TestFilterAccountsByMarket(t *testing.T) {
	accounts := []service.Account{
		{AccountID: "hk", EnabledTradingMarkets: []int{marketHongKong}},
		{AccountID: "crypto", EnabledTradingMarkets: []int{marketUnitedStates, marketCrypto}},
		{AccountID: "empty"},
	}

	filtered := filterAccountsByMarket(accounts, marketCrypto)
	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1", len(filtered))
	}
	if filtered[0].AccountID != "crypto" {
		t.Fatalf("filtered account = %#v", filtered[0])
	}
}

func TestHasTradingMarket(t *testing.T) {
	if !hasTradingMarket([]int{marketHongKong, marketCrypto}, marketCrypto) {
		t.Fatal("expected crypto market to be present")
	}
	if hasTradingMarket([]int{marketHongKong}, marketCrypto) {
		t.Fatal("expected crypto market to be absent")
	}
}

func TestResolveAssetCategory(t *testing.T) {
	restore := stubAccounts(t, `{"s":"ok","d":{"accounts":[`+
		`{"account_id":"margin","security_firm":"FUTUJP","acc_type":"margin"},`+
		`{"account_id":"deriv","security_firm":"FUTUJP","acc_type":"derivative"},`+
		`{"account_id":"cash","security_firm":"FUTUJP","acc_type":"cash"},`+
		`{"account_id":"us","security_firm":"FUTUINC","acc_type":"margin"}]}}`)
	defer restore()

	got, err := resolveAssetCategory(context.Background(), "margin", "")
	if err != nil || got != assetCategoryForeign {
		t.Fatalf("JP margin category = %q, %v", got, err)
	}
	if _, err = resolveAssetCategory(context.Background(), "deriv", ""); err == nil {
		t.Fatal("expected JP derivative account to require --asset-category")
	}
	got, err = resolveAssetCategory(context.Background(), "deriv", "domestic")
	if err != nil || got != assetCategoryDomestic {
		t.Fatalf("explicit category = %q, %v", got, err)
	}
	got, err = resolveAssetCategory(context.Background(), "cash", "")
	if err != nil || got != "" {
		t.Fatalf("JP cash category = %q, %v", got, err)
	}
	got, err = resolveAssetCategory(context.Background(), "us", "")
	if err != nil || got != "" {
		t.Fatalf("non-JP category = %q, %v", got, err)
	}
	if _, err = resolveAssetCategory(context.Background(), "us", "BOTH"); err == nil {
		t.Fatal("expected unsupported asset category to fail")
	}
}

func stubAccounts(t *testing.T, body string) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, body)
	}))
	previous := tradeSvc
	tradeSvc = service.NewTradeService(client.New(server.URL))
	return func() {
		server.Close()
		tradeSvc = previous
	}
}
