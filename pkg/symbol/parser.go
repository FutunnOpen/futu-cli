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
	MarketSG     = "SG"
	MarketJP     = "JP"
	MarketMY     = "MY"
	MarketAU     = "AU"
	MarketCA     = "CA"
	MarketKR     = "KR"
	MarketCN     = "CN"
	MarketCC     = "CC"
	MarketHKCC   = "HKCC"
	MarketBMD    = "BMD"
	MarketBMS    = "BMS"
	MarketCrypto = "CRYPTO"
)

var knownMarkets = map[string]bool{
	MarketHK:   true,
	MarketUS:   true,
	MarketSH:   true,
	MarketSZ:   true,
	MarketBJ:   true,
	MarketSG:   true,
	MarketJP:   true,
	MarketMY:   true,
	MarketAU:   true,
	MarketCA:   true,
	MarketKR:   true,
	MarketCN:   true,
	MarketCC:   true,
	MarketHKCC: true,
	MarketBMD:  true,
	MarketBMS:  true,
}

// HK stock code width for zero-padding.
const hkCodeWidth = 5

const marketCodeSeparator = "."

// Symbol represents a parsed financial instrument identifier.
type Symbol struct {
	Code   string // e.g. "00700"
	Market string // e.g. "HK"
	Full   string // e.g. "HK.00700"
}

// Parse parses a symbol input string into a Symbol struct.
//
// Supported formats:
//   - "HK.00700" -> "HK.00700"
//   - "HK.700" -> pads to "HK.00700"
//   - "US.AAPL" -> "US.AAPL"
func Parse(input string) (*Symbol, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil, fmt.Errorf("empty symbol input")
	}

	market, code, err := splitMarketCode(input)
	if err != nil {
		return nil, err
	}
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

func splitMarketCode(input string) (string, string, error) {
	market, code, ok := strings.Cut(input, marketCodeSeparator)
	if !ok || market == "" || code == "" {
		return "", "", fmt.Errorf("invalid symbol %q: expected MARKET.CODE, for example US.AAPL or HK.00700", input)
	}
	market = strings.ToUpper(market)
	if !knownMarkets[market] {
		return "", "", fmt.Errorf("invalid symbol %q: unknown market %q; expected MARKET.CODE, for example US.AAPL", input, market)
	}
	return market, code, nil
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

func normalizeCode(code, market string) string {
	if market == MarketHK && isAllDigits(code) {
		return padLeft(code, '0', hkCodeWidth)
	}
	if isAllLetters(code) && strings.ToLower(code) == code {
		return strings.ToUpper(code)
	}
	return code
}

func isAllLetters(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
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
		Full:   market + "." + code,
	}
}

// FutuCode returns the symbol in Futu API format: "MARKET.CODE"
// (e.g. "HK.09988", "US.AAPL").
func (s *Symbol) FutuCode() string {
	return s.Market + "." + s.Code
}
