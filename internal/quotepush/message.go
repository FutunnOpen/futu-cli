package quotepush

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	actionAuth      = "auth"
	actionRefresh   = "refresh"
	actionSubscribe = "subscribe"
	authTypeOAuth2  = "oauth2"
	subscribeID     = "sub-1"
	typeQuote       = "quote"
	typeOrderBook   = "order_book"
	typeTicker      = "ticker"
	typeKline       = "kline"
)

// Subscribe is the set of quote-push streams to open for the given symbols.
type Subscribe struct {
	Symbols     []string
	Types       []string
	KlinePeriod string
	KlineAdjust string
}

type authFrame struct {
	Action string        `json:"action"`
	Data   authFrameData `json:"data"`
}

type authFrameData struct {
	AuthType      string `json:"auth_type,omitempty"`
	Authorization string `json:"authorization"`
}

type subscribeFrame struct {
	ID        string     `json:"id"`
	Action    string     `json:"action"`
	Quote     []string   `json:"quote,omitempty"`
	OrderBook []string   `json:"order_book,omitempty"`
	Ticker    []string   `json:"ticker,omitempty"`
	Kline     []klineSub `json:"kline,omitempty"`
}

type klineSub struct {
	Symbol string `json:"symbol"`
	Period string `json:"period"`
	Adjust string `json:"adjust,omitempty"`
}

type inbound struct {
	Type      string
	SessionID string
	Code      *int
	Message   string
}

func authMessage(accessToken string) authFrame {
	return authFrame{
		Action: actionAuth,
		Data: authFrameData{
			AuthType:      authTypeOAuth2,
			Authorization: "Bearer " + accessToken,
		},
	}
}

func refreshMessage(accessToken string) authFrame {
	return authFrame{
		Action: actionRefresh,
		Data: authFrameData{
			Authorization: "Bearer " + accessToken,
		},
	}
}

func subscribeMessage(sub Subscribe) subscribeFrame {
	frame := subscribeFrame{ID: subscribeID, Action: actionSubscribe}
	for _, kind := range sub.Types {
		switch kind {
		case typeQuote:
			frame.Quote = append([]string(nil), sub.Symbols...)
		case typeOrderBook:
			frame.OrderBook = append([]string(nil), sub.Symbols...)
		case typeTicker:
			frame.Ticker = append([]string(nil), sub.Symbols...)
		case typeKline:
			frame.Kline = make([]klineSub, 0, len(sub.Symbols))
			for _, symbol := range sub.Symbols {
				frame.Kline = append(frame.Kline, klineSub{
					Symbol: symbol,
					Period: sub.KlinePeriod,
					Adjust: sub.KlineAdjust,
				})
			}
		}
	}
	return frame
}

func parseInbound(raw []byte) (inbound, error) {
	var payload struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
		Code      *int   `json:"code"`
		Message   string `json:"message"`
		RetCode   *int   `json:"ret_code"`
		RetMsg    string `json:"ret_msg"`
		Data      struct {
			Code    *int   `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return inbound{}, err
	}
	msg := inbound{Type: payload.Type, SessionID: payload.SessionID}
	switch {
	case payload.RetCode != nil && *payload.RetCode != 0:
		msg.Code = payload.RetCode
		msg.Message = firstText(payload.RetMsg, payload.Data.Message, payload.Message)
	case payload.Data.Code != nil && *payload.Data.Code != 0:
		msg.Code = payload.Data.Code
		msg.Message = firstText(payload.Data.Message, payload.RetMsg, payload.Message)
	case payload.Code != nil && *payload.Code != 0:
		msg.Code = payload.Code
		msg.Message = firstText(payload.Message, payload.RetMsg, payload.Data.Message)
	}
	return msg, nil
}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (msg inbound) push() bool {
	return msg.Type != ""
}

// countsTowardLimit reports whether this push uses one of the subscribed streams.
// MARKET_STATE is printed, but it does not consume --max-messages.
func (msg inbound) countsTowardLimit(sub Subscribe) bool {
	if !msg.push() {
		return false
	}
	for _, kind := range sub.Types {
		if msg.Type == pushTypeName(kind) {
			return true
		}
		if kind == typeOrderBook && msg.Type == "BROKER_QUEUE" {
			return true
		}
	}
	return false
}

func pushTypeName(kind string) string {
	switch kind {
	case typeQuote:
		return "QUOTE"
	case typeOrderBook:
		return "ORDER_BOOK"
	case typeTicker:
		return "TICKER"
	case typeKline:
		return "KLINE"
	default:
		return ""
	}
}

func (msg inbound) failed() bool {
	return msg.Code != nil && *msg.Code != 0
}

func (msg inbound) authenticated() bool {
	return msg.SessionID != "" && !msg.failed() && !msg.push()
}

// sanitizeMessage returns compact JSON safe to print. Access tokens are removed.
func sanitizeMessage(raw []byte, secret string) ([]byte, error) {
	compact := bytes.NewBuffer(make([]byte, 0, len(raw)))
	if err := json.Compact(compact, raw); err != nil {
		return nil, fmt.Errorf("quote push message is not json")
	}
	out := compact.Bytes()
	if !hasSecretKey(out) && (secret == "" || !bytes.Contains(out, []byte(secret))) {
		return out, nil
	}
	return scrubJSON(out, secret)
}

func hasSecretKey(raw []byte) bool {
	for _, key := range []string{`"authorization"`, `"access_token"`, `"refresh_token"`} {
		if bytes.Contains(raw, []byte(key)) {
			return true
		}
	}
	return false
}

func scrubJSON(raw []byte, secret string) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("quote push message is not json")
	}
	scrubValue(value)
	out, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if secret != "" {
		out = bytes.ReplaceAll(out, []byte(secret), []byte("[redacted]"))
	}
	return out, nil
}

func scrubValue(value any) {
	switch node := value.(type) {
	case map[string]any:
		delete(node, "authorization")
		delete(node, "access_token")
		delete(node, "refresh_token")
		for _, child := range node {
			scrubValue(child)
		}
	case []any:
		for _, child := range node {
			scrubValue(child)
		}
	}
}

var klinePeriods = map[string]struct{}{
	"1m": {}, "3m": {}, "5m": {}, "10m": {}, "15m": {}, "30m": {},
	"60m": {}, "120m": {}, "180m": {}, "240m": {},
	"1D": {}, "1W": {}, "1M": {}, "1Q": {}, "1Y": {},
}

var klineAdjusts = map[string]struct{}{
	"none": {}, "forward": {}, "forward_exclude_dividend": {}, "forward_include_dividend": {},
}

// NormalizeSubscribe checks symbols, stream types, and K-line parameters.
func NormalizeSubscribe(symbols, types []string, period, adjust string) (Subscribe, error) {
	if len(symbols) == 0 {
		return Subscribe{}, fmt.Errorf("at least one symbol is required")
	}
	if len(types) == 0 {
		return Subscribe{}, fmt.Errorf("at least one subscribe type is required")
	}
	if period == "" {
		period = "1m"
	}
	if adjust == "" {
		adjust = "none"
	}
	seen := make(map[string]struct{}, len(types))
	normalized := make([]string, 0, len(types))
	for _, raw := range types {
		kind := strings.ToLower(strings.TrimSpace(raw))
		switch kind {
		case typeQuote, typeOrderBook, typeTicker, typeKline:
		default:
			return Subscribe{}, fmt.Errorf("unsupported subscribe type %q", raw)
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		normalized = append(normalized, kind)
	}
	if _, ok := seen[typeKline]; ok {
		if _, valid := klinePeriods[period]; !valid {
			return Subscribe{}, fmt.Errorf("unsupported kline period %q", period)
		}
		if _, valid := klineAdjusts[adjust]; !valid {
			return Subscribe{}, fmt.Errorf("unsupported kline adjust %q", adjust)
		}
	}
	return Subscribe{
		Symbols:     symbols,
		Types:       normalized,
		KlinePeriod: period,
		KlineAdjust: adjust,
	}, nil
}
