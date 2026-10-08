package client

import "net/http"

// Endpoint describes a single API route: its HTTP method and URL path template.
type Endpoint struct {
	Method       string
	Path         string
	ReportSource bool
}

// Endpoints is the single source of truth for all Futu API routes.
// Keys follow the convention "domain.resource" or "domain.resource.action".
var Endpoints = map[string]Endpoint{
	// ── Quote ────────────────────────────────────────────────────────────
	"quote.stock_quote":  {Method: http.MethodPost, Path: "/api/v1.0/quote/stock-quote"},
	"quote.snapshot":     {Method: http.MethodGet, Path: "/v1/quote/snapshot"},
	"quote.kline":        {Method: http.MethodGet, Path: "/v1/quote/kline"},
	"quote.depth":        {Method: http.MethodGet, Path: "/v1/quote/depth"},
	"quote.ticker":       {Method: http.MethodGet, Path: "/v1/quote/ticker"},
	"quote.rank":         {Method: http.MethodGet, Path: "/v1/quote/rank"},
	"quote.option.chain": {Method: http.MethodGet, Path: "/v1/quote/option/chain"},
	"quote.static":       {Method: http.MethodGet, Path: "/v1/quote/static"},

	// ── Trade ────────────────────────────────────────────────────────────
	"trade.order.place":   {Method: http.MethodPost, Path: "/api/v1.0/accounts/{acc_id}/orders", ReportSource: true},
	"trade.order.modify":  {Method: http.MethodPut, Path: "/api/v1.0/accounts/{acc_id}/orders/{order_id}", ReportSource: true},
	"trade.order.cancel":  {Method: http.MethodDelete, Path: "/api/v1.0/accounts/{acc_id}/orders/{order_id}", ReportSource: true},
	"trade.order.confirm": {Method: http.MethodPost, Path: "/api/v1.0/accounts/{acc_id}/order_confirm", ReportSource: true},
	"trade.order.open":    {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/orders"},
	"trade.order.history": {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/orders_history"},
	"trade.order.detail":  {Method: http.MethodPost, Path: "/api/v1.0/accounts/{acc_id}/orders/detail"},
	"trade.position":      {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/positions"},
	"trade.deal.today":    {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/order_fills"},
	"trade.deal.history":  {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/fills_history"},
	"trade.account.list":  {Method: http.MethodGet, Path: "/api/v1.0/accounts/authorized_trd_accs"},
	"trade.account.funds": {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/funds"},
	"trade.account.info":  {Method: http.MethodGet, Path: "/api/v1.0/accounts/{acc_id}/acctradinginfo"},

	// ── Crypto Trading ───────────────────────────────────────────────────
	"crypto.account.total_balance": {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/total-balance"},
	"crypto.account.max_qty":       {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/purchasing-power"},
	"crypto.order.place":           {Method: http.MethodPost, Path: "/api/v1/crypto/accounts/{acc_id}/orders", ReportSource: true},
	"crypto.order.modify":          {Method: http.MethodPut, Path: "/api/v1/crypto/accounts/{acc_id}/orders/{order_id}", ReportSource: true},
	"crypto.order.cancel":          {Method: http.MethodDelete, Path: "/api/v1/crypto/accounts/{acc_id}/orders/{order_id}", ReportSource: true},
	"crypto.order.active":          {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/orders"},
	"crypto.order.history":         {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/orders/history"},
	"crypto.order.detail":          {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/orders/{order_id}"},
	"crypto.deal.fills":            {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/fills"},
	"crypto.deal.history":          {Method: http.MethodGet, Path: "/api/v1/crypto/accounts/{acc_id}/fills/history"},

	// ── Web3 ─────────────────────────────────────────────────────────────
	"web3.wallet.balance": {Method: http.MethodGet, Path: "/v1/web3/wallet/balance"},
	"web3.swap.quote":     {Method: http.MethodPost, Path: "/v1/web3/swap/quote"},
	"web3.swap.execute":   {Method: http.MethodPost, Path: "/v1/web3/swap/execute", ReportSource: true},
	"web3.transfer":       {Method: http.MethodPost, Path: "/v1/web3/transfer", ReportSource: true},
}
