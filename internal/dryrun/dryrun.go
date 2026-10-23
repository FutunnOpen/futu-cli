// Package dryrun provides a deterministic confirmation-code gate for trading
// mutations. A preview call returns a three-digit code; a second call must pass
// --execute <code> before any live order write is attempted.
package dryrun

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const (
	domainSeparator     = "futu/execute-confirmation/v1"
	codeModulus         = 1000
	codeWidth           = 3
	marketPriceLabel    = "market"
	unchangedPriceLabel = "unchanged"
	scopeFieldSeparator = "|"
	decimalBase         = 10
)

// Scope is a canonical description of the trading action being confirmed.
type Scope struct {
	canonical string
}

// Order builds a scope for a new stock or crypto order.
func Order(side, symbol, quantity, price string) Scope {
	return Scope{canonical: joinFields(
		"order",
		normalizeToken(side),
		normalizeToken(symbol),
		normalizeQuantity(quantity),
		normalizePrice(price, marketPriceLabel),
	)}
}

// OnOrder builds a scope for an action on an existing order (cancel, etc.).
func OnOrder(action, orderID string) Scope {
	return Scope{canonical: joinFields(
		"on_order",
		normalizeToken(action),
		normalizeToken(orderID),
	)}
}

// Replace builds a scope for modifying an existing order.
func Replace(orderID, quantity, price string) Scope {
	return Scope{canonical: joinFields(
		"replace",
		normalizeToken(orderID),
		normalizeQuantity(quantity),
		normalizePrice(price, unchangedPriceLabel),
	)}
}

// Code returns the three-digit confirmation code for this scope.
func (s Scope) Code() string {
	sum := sha256.Sum256([]byte(domainSeparator + "\n" + s.canonical))
	number := int(sum[0])<<8 | int(sum[1])
	return fmt.Sprintf("%0*d", codeWidth, number%codeModulus)
}

// Verify checks that code matches this scope. Leading zeros and surrounding
// whitespace are accepted.
func (s Scope) Verify(code string) error {
	trimmed := strings.TrimSpace(code)
	if len(trimmed) == 0 {
		return fmt.Errorf("confirmation code is required")
	}
	want := s.Code()
	parsed, err := strconv.ParseUint(trimmed, decimalBase, 64)
	if err != nil {
		return fmt.Errorf("invalid confirmation code %q", code)
	}
	got := fmt.Sprintf("%0*d", codeWidth, parsed%codeModulus)
	if got != want {
		return fmt.Errorf("confirmation code mismatch: got %s, want %s", got, want)
	}
	return nil
}

// Canonical returns the canonical scope string for tests and debugging.
func (s Scope) Canonical() string {
	return s.canonical
}

func joinFields(fields ...string) string {
	return strings.Join(fields, scopeFieldSeparator)
}

func normalizeToken(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeQuantity(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 {
		return ""
	}
	if normalized, ok := normalizeDecimal(trimmed); ok {
		return normalized
	}
	return normalizeToken(trimmed)
}

func normalizePrice(value, emptyLabel string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 {
		return emptyLabel
	}
	if normalized, ok := normalizeDecimal(trimmed); ok {
		return normalized
	}
	return normalizeToken(trimmed)
}

func normalizeDecimal(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 {
		return "", false
	}
	number, _, err := big.ParseFloat(trimmed, decimalBase, 0, big.ToNearestEven)
	if err != nil {
		return "", false
	}
	text := number.Text('f', -1)
	if text == "+0" || text == "-0" {
		return "0", true
	}
	return strings.TrimPrefix(text, "+"), true
}
