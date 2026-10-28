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
	{ID: "account_order_fees", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/orders/{order_id}/fees"},
	{ID: "account_margin_ratio", Method: "GET", Path: "/api/v1.0/margin_ratio"},
	{ID: "account_cash_logs", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/cash_logs"},
	{ID: "skillwrap_technical_unusual", Method: "POST", Path: "/api/v1/get_technical_unusual"},
	{ID: "skillwrap_financial_unusual", Method: "POST", Path: "/api/v1/get_financial_unusual"},
	{ID: "skillwrap_derivative_unusual", Method: "POST", Path: "/api/v1/get_derivative_unusual"},
	{ID: "account_positions", Method: "GET", Path: "/api/v1.0/accounts/{acc_id}/positions"},
	{ID: "account_authorized_trd_accs", Method: "GET", Path: "/api/v1.0/accounts/authorized_trd_accs"},
	{ID: "quote_market_snapshot", Method: "POST", Path: "/api/v1.0/quote/snapshot"},
	{ID: "quote_stock_quote", Method: "POST", Path: "/api/v1.0/quote/stock-quote"},
	{ID: "quote_option_chain", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-chain"},
	{ID: "quote_cur_kline", Method: "GET", Path: "/api/v1.0/quote/{symbol}/cur-kline"},
	{ID: "quote_history_kline", Method: "GET", Path: "/api/v1.0/quote/{symbol}/history-kline"},
	{ID: "quote_order_book", Method: "POST", Path: "/api/v1.0/quote/order-book"},
	{ID: "quote_rt_ticker", Method: "GET", Path: "/api/v1.0/quote/{symbol}/rt-ticker"},
	{ID: "quote_rt_data", Method: "GET", Path: "/api/v1.0/quote/{symbol}/rt-data"},
	{ID: "quote_stock_basicinfo", Method: "POST", Path: "/api/v1.0/quote/stock-basicinfo"},
	{ID: "quote_owner_plate", Method: "GET", Path: "/api/v1.0/quote/{symbol}/owner-plate"},
	{ID: "quote_top_ten_brokers", Method: "GET", Path: "/api/v1.0/quote/{symbol}/top-brokers"},
	{ID: "quote_top_ten_brokers_history", Method: "GET", Path: "/api/v1.0/quote/{symbol}/top-brokers-history"},
	{ID: "quote_capital_flow", Method: "GET", Path: "/api/v1.0/quote/{symbol}/capital-flow"},
	{ID: "quote_capital_flow_history", Method: "GET", Path: "/api/v1.0/quote/{symbol}/capital-flow/history"},
	{ID: "quote_capital_distribution", Method: "GET", Path: "/api/v1.0/quote/{symbol}/capital-distribution"},
	{ID: "quote_company_profile", Method: "GET", Path: "/api/v1.0/quote/{symbol}/company/profile"},
	{ID: "quote_company_executives", Method: "GET", Path: "/api/v1.0/quote/{symbol}/company/executives"},
	{ID: "quote_company_executive_background", Method: "GET", Path: "/api/v1.0/quote/{symbol}/company/executive-background"},
	{ID: "quote_company_operational_efficiency", Method: "GET", Path: "/api/v1.0/quote/{symbol}/company/operational-efficiency"},
	{ID: "quote_corporate_actions_stock_splits", Method: "GET", Path: "/api/v1.0/quote/{symbol}/corporate-actions/splits"},
	{ID: "quote_corporate_actions_buybacks", Method: "GET", Path: "/api/v1.0/quote/{symbol}/corporate-actions/buybacks"},
	{ID: "quote_corporate_actions_dividends", Method: "GET", Path: "/api/v1.0/quote/{symbol}/corporate-actions/dividends"},
	{ID: "quote_corporate_actions_rehab", Method: "GET", Path: "/api/v1.0/quote/{symbol}/corporate-actions/rehab"},
	{ID: "quote_economic_calendar_hot", Method: "GET", Path: "/api/v1.0/quote/economic-calendar/hot"},
	{ID: "quote_economic_calendar_search", Method: "GET", Path: "/api/v1.0/quote/economic-calendar/search"},
	{ID: "quote_financials_statements", Method: "GET", Path: "/api/v1.0/quote/{symbol}/financials/statements"},
	{ID: "quote_financials_revenue_breakdown", Method: "GET", Path: "/api/v1.0/quote/{symbol}/financials/revenue-breakdown"},
	{ID: "quote_financials_earnings_price_history", Method: "GET", Path: "/api/v1.0/quote/{symbol}/financials/earnings-price-history"},
	{ID: "quote_financials_earnings_price_move", Method: "GET", Path: "/api/v1.0/quote/{symbol}/financials/earnings-price-move"},
	{ID: "quote_future_info", Method: "POST", Path: "/api/v1.0/quote/future-info"},
	{ID: "quote_ipo_list_hk", Method: "GET", Path: "/api/v1.0/quote/ipo-list/hk"},
	{ID: "quote_ipo_list_us", Method: "GET", Path: "/api/v1.0/quote/ipo-list/us"},
	{ID: "quote_ipo_list_sg", Method: "GET", Path: "/api/v1.0/quote/ipo-list/sg"},
	{ID: "quote_ipo_list_my", Method: "GET", Path: "/api/v1.0/quote/ipo-list/my"},
	{ID: "quote_ipo_list_cn", Method: "GET", Path: "/api/v1.0/quote/ipo-list/cn"},
	{ID: "quote_market_state", Method: "POST", Path: "/api/v1.0/quote/market-state"},
	{ID: "quote_option_expiration_date", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-expiration"},
	{ID: "quote_option_strategy", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-strategy"},
	{ID: "quote_option_strategy_spread", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-strategy-spread"},
	{ID: "quote_combo_option_quote", Method: "POST", Path: "/api/v1.0/quote/combo-option-quote"},
	{ID: "quote_option_exercise_probability", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-exercise-probability"},
	{ID: "quote_option_volatility", Method: "GET", Path: "/api/v1.0/quote/{symbol}/option-volatility"},
	{ID: "quote_plate_list", Method: "GET", Path: "/api/v1.0/quote/plate-list"},
	{ID: "quote_plate_stock", Method: "GET", Path: "/api/v1.0/quote/plate-stock"},
	{ID: "quote_referencefuture_list", Method: "GET", Path: "/api/v1.0/quote/{symbol}/reference-future"},
	{ID: "quote_research_morningstar_report", Method: "GET", Path: "/api/v1.0/quote/{symbol}/research/morningstar"},
	{ID: "quote_research_analyst_consensus", Method: "GET", Path: "/api/v1.0/quote/{symbol}/research/analyst-consensus"},
	{ID: "quote_research_rating_summary", Method: "GET", Path: "/api/v1.0/quote/{symbol}/research/rating-summary"},
	{ID: "quote_stock_screen", Method: "POST", Path: "/api/v1.0/quote/stock-screen"},
	{ID: "quote_warrant_screen", Method: "POST", Path: "/api/v1.0/quote/warrant-screen"},
	{ID: "quote_option_screen", Method: "POST", Path: "/api/v1.0/quote/option-screen"},
	{ID: "quote_news_search", Method: "GET", Path: "/api/v1.0/quote/find-news"},
	{ID: "quote_community_search", Method: "GET", Path: "/api/v1.0/quote/find-community"},
	{ID: "quote_stock_feed", Method: "GET", Path: "/api/v1.0/quote/find-stock-feed"},
	{ID: "quote_shareholders_overview", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/overview"},
	{ID: "quote_shareholders_institutional", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/institutional"},
	{ID: "quote_shareholders_holding_changes", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/holding-changes"},
	{ID: "quote_shareholders_holder_detail", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/holder-detail"},
	{ID: "quote_insider_holder_list", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/insider-holders"},
	{ID: "quote_insider_trade_list", Method: "GET", Path: "/api/v1.0/quote/{symbol}/shareholders/insider-trades"},
	{ID: "quote_daily_short_volume", Method: "GET", Path: "/api/v1.0/quote/{symbol}/short/daily-volume"},
	{ID: "quote_short_interest", Method: "GET", Path: "/api/v1.0/quote/{symbol}/short/interest"},
	{ID: "quote_trading_days", Method: "GET", Path: "/api/v1.0/quote/trading-days"},
	{ID: "quote_user_security_group", Method: "GET", Path: "/api/v1.0/quote/user-security-group"},
	{ID: "quote_user_security", Method: "GET", Path: "/api/v1.0/quote/user-security"},
	{ID: "quote_modify_user_security", Method: "POST", Path: "/api/v1.0/quote/modify-user-security"},
	{ID: "quote_valuation_detail", Method: "GET", Path: "/api/v1.0/quote/{symbol}/valuation/detail"},
	{ID: "quote_valuation_plate_stock_list", Method: "GET", Path: "/api/v1.0/quote/valuation/plate-stocks"},
	{ID: "quote_valuation_index_component_stock_list", Method: "GET", Path: "/api/v1.0/quote/valuation/index-stocks"},
	{ID: "quote_valuation_index_stock_plate_list", Method: "GET", Path: "/api/v1.0/quote/valuation/index-stock-plates"},
	{ID: "sim_trade_account_list", Method: "GET", Path: "/api/v1.0/sim-trade/accounts"},
	{ID: "sim_trade_cash_info", Method: "GET", Path: "/api/v1.0/sim-trade/{acc_id}/cash-info"},
	{ID: "sim_trade_order_list", Method: "GET", Path: "/api/v1.0/sim-trade/{acc_id}/orders"},
	{ID: "sim_trade_history_order_list", Method: "GET", Path: "/api/v1.0/sim-trade/{acc_id}/history-orders"},
	{ID: "sim_trade_max_buy_sell", Method: "GET", Path: "/api/v1.0/sim-trade/{acc_id}/max-buy-sell"},
	{ID: "sim_trade_input_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders"},
	{ID: "sim_trade_cancel_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders/{order_id}/cancel"},
	{ID: "sim_trade_modify_order", Method: "POST", Path: "/api/v1.0/sim-trade/{acc_id}/orders/{order_id}/modify"},
	{ID: "sim_trade_position_list", Method: "GET", Path: "/api/v1.0/sim-trade/{acc_id}/positions"},
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
