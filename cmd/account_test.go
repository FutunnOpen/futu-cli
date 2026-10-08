package cmd

import (
	"testing"

	"github.com/FutunnOpen/futu-cli/internal/service"
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
