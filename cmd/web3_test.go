package cmd

import (
	"strings"
	"testing"
)

func TestParseWeb3AmountPreservesHighPrecision(t *testing.T) {
	const amount = "0.123456789012345678901234567890"

	got, err := parseWeb3Amount("  " + amount + "  ")
	if err != nil {
		t.Fatalf("parseWeb3Amount() error = %v", err)
	}
	if got != amount {
		t.Fatalf("amount = %q, want %q", got, amount)
	}
}

func TestParseWeb3AmountRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "0", "0.0", "-1", "+1", "1e-8", ".1", "1.", "1.2.3", "NaN"} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseWeb3Amount(value); err == nil {
				t.Fatalf("parseWeb3Amount(%q) succeeded", value)
			}
		})
	}
}

func TestConfirmWeb3ActionWithReader(t *testing.T) {
	if !confirmWeb3ActionWithReader("Confirm?", strings.NewReader("yes\n")) {
		t.Fatal("yes should confirm")
	}
	if confirmWeb3ActionWithReader("Confirm?", strings.NewReader("no\n")) {
		t.Fatal("no should cancel")
	}
	if confirmWeb3ActionWithReader("Confirm?", strings.NewReader("yikes\n")) {
		t.Fatal("only y or yes should confirm")
	}
}
