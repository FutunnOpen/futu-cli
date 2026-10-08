package reporting

import (
	"net/url"
	"strings"
)

type ToolRoute struct {
	ID     string
	Method string
	Path   string
}

var toolRoutes = []ToolRoute{
	{ID: "trading_order_place", Method: "POST", Path: "/api/v1.0/accounts/{acc_id}/orders"},
	{ID: "trading_order_replace", Method: "PUT", Path: "/api/v1.0/accounts/{acc_id}/orders/{order_id}"},
	{ID: "trading_order_cancel", Method: "DELETE", Path: "/api/v1.0/accounts/{acc_id}/orders/{order_id}"},
	{ID: "trading_order_confirm", Method: "POST", Path: "/api/v1.0/accounts/{acc_id}/order_confirm"},
	{ID: "account_orders_active", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/orders"},
	{ID: "account_orders_history", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/orders_history"},
	{ID: "account_orders_detail", Method: "POST", Path: "/api/v1.0/accounts/{acc_id}/orders/detail"},
	{ID: "account_order_fills_today", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/order_fills"},
	{ID: "account_fills_history", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/fills_history"},
	{ID: "account_funds", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/funds"},
	{ID: "account_trading_info", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/acctradinginfo"},
	{ID: "account_positions", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/positions"},
	{ID: "account_authorized_trd_accs", Method: "GET", Path: "/api/v1.0/accounts/authorized_trd_accs"},
	{ID: "quote_market_snapshot", Method: "POST", Path: "/api/v1.0/quote/snapshot"},
	{ID: "quote_stock_quote", Method: "POST", Path: "/api/v1.0/quote/stock-quote"},
	{ID: "quote_option_chain", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-chain"},
	{ID: "sim_trade_input_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders"},
	{ID: "sim_trade_cancel_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders/{order_id}/cancel"},
	{ID: "sim_trade_modify_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders/{order_id}/modify"},
	{ID: "crypto_account_total_balance", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/total-balance"},
	{ID: "crypto_account_purchasing_power", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/purchasing-power"},
	{ID: "crypto_account_modify_order", Method: "PUT", Path: "/api/v1/crypto/accounts/{acc_id}/orders/{orderId}"},
	{ID: "crypto_account_create_order", Method: "POST", Path: "/api/v1/crypto/accounts/{acc_id}/orders"},
	{ID: "crypto_account_cancel_order", Method: "DELETE", Path: "/api/v1/crypto/accounts/{acc_id}/orders/{orderId}"},
	{ID: "crypto_account_active_orders", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/orders"},
	{ID: "crypto_account_order_detail", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/orders/{orderId}"},
	{ID: "crypto_account_order_history", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/orders/history"},
	{ID: "crypto_account_fills", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/fills"},
	{ID: "crypto_account_fills_history", Method: "GET", Path: "/api/v1/crypto/accounts/{acc_id}/fills/history"},
}

func MatchToolRoute(method, path string) ToolRoute {
	method = strings.ToUpper(method)
	path = normalizePath(path)
	for _, route := range toolRoutes {
		if route.Method != method {
			continue
		}
		if matchPathTemplate(route.Path, path) {
			return route
		}
	}
	return ToolRoute{}
}

func ShouldCoarseRedact(toolID, path string) bool {
	if strings.HasPrefix(toolID, "sim_trade_") || strings.Contains(path, "/sim-trade/") {
		return false
	}
	prefixes := []string{
		"trading_",
		"account_order",
		"account_orders",
		"account_fills",
		"account_funds",
		"account_positions",
		"account_trading_info",
		"crypto_account_",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(toolID, prefix) {
			return true
		}
	}
	return strings.Contains(path, "/crypto/") ||
		strings.Contains(path, "/web3/") ||
		isAccountSensitivePath(path)
}

func isAccountSensitivePath(path string) bool {
	if !strings.Contains(path, "/accounts/") {
		return false
	}
	sensitiveSegments := []string{
		"/orders",
		"/orders_history",
		"/order_fills",
		"/fills_history",
		"/funds",
		"/positions",
		"/acctradinginfo",
	}
	for _, segment := range sensitiveSegments {
		if strings.Contains(path, segment) {
			return true
		}
	}
	return false
}

func matchPathTemplate(template, path string) bool {
	templateParts := splitPath(template)
	pathParts := splitPath(path)
	if len(templateParts) != len(pathParts) {
		return false
	}
	for index, templatePart := range templateParts {
		if isPlaceholder(templatePart) || isPlaceholder(pathParts[index]) {
			continue
		}
		if templatePart != pathParts[index] {
			return false
		}
	}
	return true
}

func normalizePath(path string) string {
	if parsed, err := url.Parse(path); err == nil {
		path = parsed.Path
	}
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func splitPath(path string) []string {
	path = strings.Trim(normalizePath(path), "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func isPlaceholder(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}
