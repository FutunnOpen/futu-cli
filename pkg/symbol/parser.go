package symbol

import (
	"fmt"
	"strings"
	"unicode"
)

// Market identifier constants.
const (
	MarketHK     = "HK"
	MarketUS     = "US"
	MarketSH     = "SH"
	MarketSZ     = "SZ"
	MarketBJ     = "BJ"
	MarketBMD    = "BMD"
	MarketCrypto = "CRYPTO"
)

// knownMarkets contains market prefixes recognised by the Futu API.
var knownMarkets = map[string]bool{
	MarketHK:  true,
	MarketUS:  true,
	MarketSH:  true,
	MarketSZ:  true,
	MarketBJ:  true,
	MarketBMD: true,
}

// HK stock code width for zero-padding.
const hkCodeWidth = 5

// Symbol represents a parsed financial instrument identifier.
type Symbol struct {
	Code   string // e.g. "00700"
	Market string // e.g. "HK"
	Full   string // e.g. "00700.HK"
}

// Parse parses a symbol input string into a Symbol struct.
//
// Supported formats:
//   - "700.HK" -> pads to "00700.HK"
//   - "AAPL" -> defaults to "AAPL.US"
//   - "AAPL.US" -> as-is
//   - "BTC-USD" -> "BTC-USD.CRYPTO"
func Parse(input string) (*Symbol, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil, fmt.Errorf("empty symbol input")
	}

	if isCryptoSymbol(input) {
		return buildCrypto(input), nil
	}

	code, market := splitSymbol(input)
	if len(market) == 0 {
		market = inferMarket(code)
	}
	market = strings.ToUpper(market)
	code = normalizeCode(code, market)

	return buildSymbol(code, market), nil
}

// ParseMulti parses multiple symbol input strings.
func ParseMulti(inputs []string) ([]*Symbol, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no symbol inputs provided")
	}

	symbols := make([]*Symbol, 0, len(inputs))
	for _, input := range inputs {
		sym, err := Parse(input)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", input, err)
		}
		symbols = append(symbols, sym)
	}
	return symbols, nil
}

// isCryptoSymbol checks if the input looks like a crypto pair (contains "-"
// but no ".").
func isCryptoSymbol(input string) bool {
	return strings.Contains(input, "-") && !strings.Contains(input, ".")
}

// buildCrypto builds a Symbol for a crypto pair.
func buildCrypto(input string) *Symbol {
	code := strings.ToUpper(input)
	return buildSymbol(code, MarketCrypto)
}

// splitSymbol splits a dotted symbol into (code, market).
// Supports both "CODE.MARKET" (e.g. "700.HK") and Futu-format
// "MARKET.CODE" (e.g. "HK.09988") by checking whether the left
// side is a known market prefix.
func splitSymbol(input string) (string, string) {
	idx := strings.LastIndex(input, ".")
	if idx < 0 {
		return input, ""
	}
	left, right := input[:idx], input[idx+1:]
	if knownMarkets[strings.ToUpper(left)] {
		return right, left
	}
	return left, right
}

// inferMarket determines the default market based on the code format.
// Pure digits default to HK; anything else defaults to US.
func inferMarket(code string) string {
	if isAllDigits(code) {
		return MarketHK
	}
	return MarketUS
}

// isAllDigits returns true if s consists entirely of ASCII digits.
func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// normalizeCode applies market-specific normalization.
// HK stock codes are zero-padded to 5 digits.
func normalizeCode(code, market string) string {
	if market == MarketHK && isAllDigits(code) {
		return padLeft(code, '0', hkCodeWidth)
	}
	return strings.ToUpper(code)
}

// padLeft pads s with the given character on the left until it reaches width.
func padLeft(s string, pad rune, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(string(pad), width-len(s)) + s
}

// buildSymbol constructs a Symbol from code and market.
func buildSymbol(code, market string) *Symbol {
	return &Symbol{
		Code:   code,
		Market: market,
		Full:   code + "." + market,
	}
}

// FutuCode returns the symbol in Futu API format: "MARKET.CODE"
// (e.g. "HK.09988", "US.AAPL").
func (s *Symbol) FutuCode() string {
	return s.Market + "." + s.Code
}
