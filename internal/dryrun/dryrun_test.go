package dryrun

import (
	"strings"
	"testing"
)

func TestOrderCanonicalizesDecimalAndSymbol(t *testing.T) {
	left := Order("buy", "700.hk", "400", "400.00")
	right := Order("BUY", "700.HK", "4e2", "+400.0")
	if left.Canonical() != right.Canonical() {
		t.Fatalf("canonical mismatch: %q vs %q", left.Canonical(), right.Canonical())
	}
	if left.Code() != right.Code() {
		t.Fatalf("code mismatch: %s vs %s", left.Code(), right.Code())
	}
	if !strings.HasPrefix(left.Canonical(), "order|BUY|700.HK|400|400") {
		t.Fatalf("canonical = %q", left.Canonical())
	}
}

func TestOrderUsesMarketLabelForEmptyPrice(t *testing.T) {
	scope := Order("BUY", "US.AAPL", "1", "")
	if !strings.HasSuffix(scope.Canonical(), "|market") {
		t.Fatalf("canonical = %q", scope.Canonical())
	}
}

func TestReplaceUsesUnchangedLabelForEmptyPrice(t *testing.T) {
	scope := Replace("ORD1", "2", "")
	if !strings.HasSuffix(scope.Canonical(), "|unchanged") {
		t.Fatalf("canonical = %q", scope.Canonical())
	}
}

func TestCodeIsThreeDigitsAndSelfVerifies(t *testing.T) {
	scope := Order("BUY", "US.AAPL", "1", "100.5")
	code := scope.Code()
	if len(code) != codeWidth {
		t.Fatalf("code length = %d, want %d", len(code), codeWidth)
	}
	if err := scope.Verify(code); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := scope.Verify("  " + code + " "); err != nil {
		t.Fatalf("Verify(trimmed) error = %v", err)
	}
	if err := scope.Verify("0" + code); err != nil {
		t.Fatalf("Verify(leading zero) error = %v", err)
	}
	if code != "999" {
		if err := scope.Verify("999"); err == nil {
			t.Fatal("Verify() accepted mismatched code")
		}
	}
}

func TestOnOrderCancelScope(t *testing.T) {
	scope := OnOrder("cancel", "ord-1")
	if scope.Canonical() != "on_order|CANCEL|ORD-1" {
		t.Fatalf("canonical = %q", scope.Canonical())
	}
	if err := scope.Verify(scope.Code()); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyRejectsEmptyAndInvalid(t *testing.T) {
	scope := Order("BUY", "US.AAPL", "1", "10")
	if err := scope.Verify(""); err == nil {
		t.Fatal("expected empty code error")
	}
	if err := scope.Verify("abc"); err == nil {
		t.Fatal("expected invalid code error")
	}
}
