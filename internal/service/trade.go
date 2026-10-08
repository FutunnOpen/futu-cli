package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

// Endpoint names for trade API routes.
const (
	endpointOrderPlace   = "trade.order.place"
	endpointOrderModify  = "trade.order.modify"
	endpointOrderCancel  = "trade.order.cancel"
	endpointOrderConfirm = "trade.order.confirm"
	endpointOrderOpen    = "trade.order.open"
	endpointOrderHistory = "trade.order.history"
	endpointOrderDetail  = "trade.order.detail"
	endpointPosition     = "trade.position"
	endpointDealToday    = "trade.deal.today"
	endpointDealHistory  = "trade.deal.history"
	endpointAccountList  = "trade.account.list"
	endpointAccountFund  = "trade.account.funds"
	endpointAccountInfo  = "trade.account.info"
	accountAPIStatusOK   = "ok"

	DefaultTradingMarket = "HK"
	DefaultPageSize      = 50
	minPageSize          = 10
	maxPageSize          = 100
)

// TradeService provides access to trading and account endpoints.
type TradeService struct {
	client *client.Client
}

// NewTradeService creates a new TradeService backed by the given client.
func NewTradeService(c *client.Client) *TradeService {
	return &TradeService{client: c}
}

// --- Request / response types ---

// Order represents a trading order.
type Order struct {
	OrderID    string         `json:"order_id"`
	Symbol     string         `json:"symbol"`
	Side       string         `json:"side"` // "BUY"/"SELL"
	Type       string         `json:"type"` // "LIMIT"/"MARKET"
	Price      float64        `json:"price"`
	Qty        int64          `json:"qty"`
	FilledQty  int64          `json:"filled_qty"`
	Status     string         `json:"status"`
	CreateTime int64          `json:"create_time"`
	Extra      map[string]any `json:"-"`
}

type orderWire struct {
	OrderID     string          `json:"order_id"`
	Symbol      string          `json:"symbol"`
	Code        string          `json:"code"`
	Side        string          `json:"side"`
	TradeSide   string          `json:"trd_side"`
	Type        string          `json:"type"`
	OrderType   string          `json:"order_type"`
	Price       json.RawMessage `json:"price"`
	Qty         json.RawMessage `json:"qty"`
	FilledQty   json.RawMessage `json:"filled_qty"`
	DealtQty    json.RawMessage `json:"dealt_qty"`
	Status      string          `json:"status"`
	OrderStatus string          `json:"order_status"`
	CreateTime  int64           `json:"create_time"`
}

// UnmarshalJSON accepts both the legacy CLI shape and the official trading API
// order shape documented in docs/trading-naming-dictionary.md.
func (o *Order) UnmarshalJSON(data []byte) error {
	var wire orderWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	price, err := decodeFloat(wire.Price)
	if err != nil {
		return fmt.Errorf("decode price: %w", err)
	}
	qty, err := decodeInt64(wire.Qty)
	if err != nil {
		return fmt.Errorf("decode qty: %w", err)
	}
	filledQty, err := decodeInt64(firstRaw(wire.FilledQty, wire.DealtQty))
	if err != nil {
		return fmt.Errorf("decode filled quantity: %w", err)
	}

	*o = Order{
		OrderID:    wire.OrderID,
		Symbol:     firstString(wire.Symbol, wire.Code),
		Side:       firstString(wire.Side, wire.TradeSide),
		Type:       firstString(wire.Type, wire.OrderType),
		Price:      price,
		Qty:        qty,
		FilledQty:  filledQty,
		Status:     firstString(wire.Status, wire.OrderStatus),
		CreateTime: wire.CreateTime,
		Extra:      extraOrderFields(data),
	}
	return nil
}

func (o Order) MarshalJSON() ([]byte, error) {
	values := make(map[string]any, len(o.Extra)+orderJSONBaseFieldCount)
	for key, value := range o.Extra {
		values[key] = value
	}
	values["order_id"] = o.OrderID
	values["symbol"] = o.Symbol
	values["side"] = o.Side
	values["type"] = o.Type
	values["price"] = o.Price
	values["qty"] = o.Qty
	values["filled_qty"] = o.FilledQty
	values["status"] = o.Status
	values["create_time"] = o.CreateTime
	return json.Marshal(values)
}

const (
	qtyFormatBase           = 10
	bitSize64               = 64
	orderTypeModified       = "MODIFIED"
	orderJSONBaseFieldCount = 9
	dealJSONBaseFieldCount  = 12
)

var canonicalOrderFields = map[string]bool{
	"order_id":     true,
	"symbol":       true,
	"code":         true,
	"side":         true,
	"trd_side":     true,
	"type":         true,
	"order_type":   true,
	"price":        true,
	"qty":          true,
	"filled_qty":   true,
	"dealt_qty":    true,
	"status":       true,
	"order_status": true,
	"create_time":  true,
}

var canonicalDealFields = map[string]bool{
	"deal_id":             true,
	"order_id":            true,
	"symbol":              true,
	"code":                true,
	"stock_name":          true,
	"side":                true,
	"trd_side":            true,
	"price":               true,
	"qty":                 true,
	"time":                true,
	"create_time":         true,
	"updated_time":        true,
	"status":              true,
	"counter_broker_id":   true,
	"counter_broker_name": true,
}

// PlaceOrderRequest is the payload for placing a new order.
type PlaceOrderRequest struct {
	Account      string          `json:"-"`
	Code         string          `json:"code"`
	Side         string          `json:"side"`
	OrderType    string          `json:"order_type"`
	Price        string          `json:"-"`
	Qty          int64           `json:"-"`
	TimeInForce  string          `json:"time_in_force"`
	LotType      string          `json:"lot_type,omitempty"`
	Remark       string          `json:"remark,omitempty"`
	Session      string          `json:"session,omitempty"`
	AuxPrice     string          `json:"-"`
	OrderClass   string          `json:"order_class,omitempty"`
	MultiLegInfo json.RawMessage `json:"multi_leg_info,omitempty"`
}

type placeOrderPayload struct {
	Code         string          `json:"code,omitempty"`
	Side         string          `json:"side"`
	OrderType    string          `json:"order_type"`
	Qty          string          `json:"qty"`
	Price        string          `json:"price,omitempty"`
	TimeInForce  string          `json:"time_in_force"`
	LotType      string          `json:"lot_type,omitempty"`
	Remark       string          `json:"remark,omitempty"`
	Session      string          `json:"session,omitempty"`
	AuxPrice     string          `json:"aux_price,omitempty"`
	OrderClass   string          `json:"order_class,omitempty"`
	MultiLegInfo json.RawMessage `json:"multi_leg_info,omitempty"`
}

// ModifyOrderRequest is the payload for modifying an existing order.
type ModifyOrderRequest struct {
	Account  string `json:"-"`
	Exchange string `json:"exchange"`
	Qty      int64  `json:"-"`
	Price    string `json:"-"`
	AuxPrice string `json:"-"`
}

type modifyOrderPayload struct {
	Exchange string `json:"exchange"`
	Qty      string `json:"qty"`
	Price    string `json:"price,omitempty"`
	AuxPrice string `json:"aux_price,omitempty"`
}

// CancelOrderRequest is the payload for cancelling an existing order.
type CancelOrderRequest struct {
	Account  string `json:"-"`
	Exchange string `json:"exchange"`
}

type OrderConfirmRequest struct {
	Account   string
	ConfirmID string
	Exchange  string
}

type orderConfirmPayload struct {
	ConfirmID string `json:"confirm_id"`
	Exchange  string `json:"exchange,omitempty"`
}

// TradingInfoRequest is the query for maximum tradable quantities.
type TradingInfoRequest struct {
	Account   string
	Code      string
	OrderType string
	Price     string
	OrderID   string
}

// OpenOrdersRequest holds parameters for querying open (incomplete) orders.
type OpenOrdersRequest struct {
	Account  string
	Market   string
	PageSize int
}

// HistoryOrdersRequest holds parameters for querying historical orders.
type HistoryOrdersRequest struct {
	Account  string
	Market   string
	Start    int64
	End      int64
	Symbol   string
	PageSize int
}

// OrderDetailsRequest holds parameters for querying order details.
type OrderDetailsRequest struct {
	Account  string
	Exchange string
	OrderIDs []string
}

type orderDetailsPayload struct {
	OrderIDs []string `json:"order_ids"`
	Exchange string   `json:"exchange"`
}

type OrderConfirmationRequiredError struct {
	Account   string
	ConfirmID string
	Exchange  string
}

func (e *OrderConfirmationRequiredError) Error() string {
	return "order confirmation required"
}

// Position side constants returned by the positions API.
const (
	PositionSideNone  = "NONE"
	PositionSideLong  = "LONG"
	PositionSideShort = "SHORT"
)

// Position represents a held position in a security.
type Position struct {
	PositionSide   string  `json:"position_side"`
	Code           string  `json:"code"`
	StockName      string  `json:"stock_name"`
	Qty            Decimal `json:"qty"`
	CanSellQty     Decimal `json:"can_sell_qty"`
	Currency       string  `json:"currency"`
	NominalPrice   Decimal `json:"nominal_price"`
	CostPrice      Decimal `json:"cost_price"`
	CostPriceValid bool    `json:"cost_price_valid"`
	MarketVal      Decimal `json:"market_val"`
	PLRatio        Decimal `json:"pl_ratio"`
	PLRatioValid   bool    `json:"pl_ratio_valid"`
	PLVal          Decimal `json:"pl_val"`
	PLValValid     bool    `json:"pl_val_valid"`
	TodayPLVal     Decimal `json:"today_pl_val"`
	TodayTrdVal    Decimal `json:"today_trd_val"`
	TodayBuyQty    Decimal `json:"today_buy_qty"`
	TodayBuyVal    Decimal `json:"today_buy_val"`
	TodaySellQty   Decimal `json:"today_sell_qty"`
	TodaySellVal   Decimal `json:"today_sell_val"`
	UnrealizedPL   Decimal `json:"unrealized_pl"`
	RealizedPL     Decimal `json:"realized_pl"`
}

// PositionFilter holds optional query parameters for the positions API.
type PositionFilter struct {
	Code       string
	PLRatioMin string
	PLRatioMax string
}

// Deal represents a single executed trade.
type Deal struct {
	DealID            string         `json:"deal_id"`
	OrderID           string         `json:"order_id"`
	Symbol            string         `json:"symbol"`
	StockName         string         `json:"stock_name"`
	Side              string         `json:"side"`
	Price             float64        `json:"price"`
	Qty               int64          `json:"qty"`
	Time              int64          `json:"time"`
	UpdatedTime       int64          `json:"updated_time"`
	Status            string         `json:"status"`
	CounterBrokerID   int64          `json:"counter_broker_id"`
	CounterBrokerName string         `json:"counter_broker_name"`
	Extra             map[string]any `json:"-"`
}

// TodayDealsRequest holds parameters for querying today's fills.
type TodayDealsRequest struct {
	Account  string
	Market   string
	PageSize int
}

// HistoryDealsRequest holds parameters for querying historical fills.
type HistoryDealsRequest struct {
	Account  string
	Market   string
	Start    int64
	End      int64
	Symbol   string
	PageSize int
}

type dealWire struct {
	DealID            string          `json:"deal_id"`
	OrderID           string          `json:"order_id"`
	Symbol            string          `json:"symbol"`
	Code              string          `json:"code"`
	StockName         string          `json:"stock_name"`
	Side              string          `json:"side"`
	TradeSide         string          `json:"trd_side"`
	Price             json.RawMessage `json:"price"`
	Qty               json.RawMessage `json:"qty"`
	Time              int64           `json:"time"`
	CreateTime        int64           `json:"create_time"`
	UpdatedTime       int64           `json:"updated_time"`
	Status            string          `json:"status"`
	CounterBrokerID   json.RawMessage `json:"counter_broker_id"`
	CounterBrokerName string          `json:"counter_broker_name"`
}

func (d *Deal) UnmarshalJSON(data []byte) error {
	var wire dealWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	price, err := decodeFloat(wire.Price)
	if err != nil {
		return fmt.Errorf("decode deal price: %w", err)
	}
	qty, err := decodeInt64(wire.Qty)
	if err != nil {
		return fmt.Errorf("decode deal qty: %w", err)
	}
	brokerID, err := decodeInt64(wire.CounterBrokerID)
	if err != nil {
		return fmt.Errorf("decode counter broker id: %w", err)
	}
	*d = Deal{
		DealID:            wire.DealID,
		OrderID:           wire.OrderID,
		Symbol:            firstString(wire.Symbol, wire.Code),
		StockName:         wire.StockName,
		Side:              firstString(wire.Side, wire.TradeSide),
		Price:             price,
		Qty:               qty,
		Time:              firstInt64(wire.Time, wire.CreateTime),
		UpdatedTime:       wire.UpdatedTime,
		Status:            wire.Status,
		CounterBrokerID:   brokerID,
		CounterBrokerName: wire.CounterBrokerName,
		Extra:             extraDealFields(data),
	}
	return nil
}

func (d Deal) MarshalJSON() ([]byte, error) {
	values := make(map[string]any, len(d.Extra)+dealJSONBaseFieldCount)
	for key, value := range d.Extra {
		values[key] = value
	}
	values["deal_id"] = d.DealID
	values["order_id"] = d.OrderID
	values["symbol"] = d.Symbol
	values["stock_name"] = d.StockName
	values["side"] = d.Side
	values["price"] = d.Price
	values["qty"] = d.Qty
	values["time"] = d.Time
	values["updated_time"] = d.UpdatedTime
	values["status"] = d.Status
	values["counter_broker_id"] = d.CounterBrokerID
	values["counter_broker_name"] = d.CounterBrokerName
	return json.Marshal(values)
}

// Account represents a trading account.
type Account struct {
	FutuID                string `json:"futu_id"`
	AccountID             string `json:"account_id"`
	AccountNo             string `json:"account_card_number"`
	UniversalAccountNo    string `json:"univs_account_card_number"`
	MemberID              string `json:"member_id"`
	Type                  string `json:"acc_type"`
	Channel               string `json:"channel"`
	AccountChannel        string `json:"account_channel"`
	SecurityFirm          string `json:"security_firm"`
	Name                  string `json:"name"`
	EnabledTradingMarkets []int  `json:"enable_market"`
}

type accountWire struct {
	FutuID                string          `json:"futu_id"`
	AccountID             json.RawMessage `json:"account_id"`
	AccountNo             json.RawMessage `json:"account_card_number"`
	UniversalAccountNo    json.RawMessage `json:"univs_account_card_number"`
	MemberID              string          `json:"member_id"`
	Type                  string          `json:"acc_type"`
	Channel               string          `json:"channel"`
	AccountChannel        string          `json:"account_channel"`
	SecurityFirm          string          `json:"security_firm"`
	Name                  string          `json:"name"`
	EnabledTradingMarkets []int           `json:"enable_market"`
}

// UnmarshalJSON accepts IDs encoded as either JSON strings or numbers.
func (a *Account) UnmarshalJSON(data []byte) error {
	var wire accountWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	accountID, err := decodeStringOrNumber(wire.AccountID)
	if err != nil {
		return fmt.Errorf("decode account_id: %w", err)
	}
	accountNo, err := decodeStringOrNumber(wire.AccountNo)
	if err != nil {
		return fmt.Errorf("decode account_card_number: %w", err)
	}
	universalAccountNo, err := decodeStringOrNumber(wire.UniversalAccountNo)
	if err != nil {
		return fmt.Errorf("decode univs_account_card_number: %w", err)
	}
	*a = Account{
		FutuID: wire.FutuID, AccountID: accountID, AccountNo: accountNo,
		UniversalAccountNo: universalAccountNo, MemberID: wire.MemberID,
		Type: wire.Type, Channel: wire.Channel,
		AccountChannel: wire.AccountChannel, SecurityFirm: wire.SecurityFirm,
		Name: wire.Name, EnabledTradingMarkets: wire.EnabledTradingMarkets,
	}
	return nil
}

func decodeStringOrNumber(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", err
		}
		return value, nil
	}
	var value json.Number
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value.String(), nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return ""
}

func firstInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstDealSlice(values ...[]Deal) []Deal {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 && string(value) != "null" {
			return value
		}
	}
	return nil
}

func extraOrderFields(data []byte) map[string]any {
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil
	}
	for key := range canonicalOrderFields {
		delete(values, key)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func extraDealFields(data []byte) map[string]any {
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil
	}
	for key := range canonicalDealFields {
		delete(values, key)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func decodeFloat(raw json.RawMessage) (float64, error) {
	value, err := decodeStringOrNumber(raw)
	if err != nil {
		return 0, err
	}
	if len(strings.TrimSpace(value)) == 0 {
		return 0, nil
	}
	return strconv.ParseFloat(value, bitSize64)
}

func decodeInt64(raw json.RawMessage) (int64, error) {
	value, err := decodeStringOrNumber(raw)
	if err != nil {
		return 0, err
	}
	if len(strings.TrimSpace(value)) == 0 {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, bitSize64)
	if err != nil {
		return 0, err
	}
	return int64(parsed), nil
}

// Funds represents the monetary summary of a trading account.
type Funds struct {
	Power                 Decimal `json:"power"`
	MaxPowerShort         Decimal `json:"max_power_short"`
	TotalAssets           Decimal `json:"total_assets"`
	SecuritiesAssets      Decimal `json:"securities_assets"`
	FundsAssets           Decimal `json:"funds_assets"`
	Cash                  Decimal `json:"cash"`
	MarketValue           Decimal `json:"market_val"`
	LongMarketValue       Decimal `json:"long_mv"`
	ShortMarketValue      Decimal `json:"short_mv"`
	PendingAsset          Decimal `json:"pending_asset"`
	FrozenCash            Decimal `json:"frozen_cash"`
	MaxWithdrawal         Decimal `json:"max_withdrawal"`
	Currency              string  `json:"currency"`
	AvailableFunds        Decimal `json:"available_funds"`
	UnrealizedPL          Decimal `json:"unrealized_pl"`
	RealizedPL            Decimal `json:"realized_pl"`
	RiskStatus            string  `json:"risk_status"`
	InitialMargin         Decimal `json:"initial_margin"`
	MarginCallMargin      Decimal `json:"margin_call_margin"`
	MaintenanceMargin     Decimal `json:"maintenance_margin"`
	HKCash                Decimal `json:"hk_cash"`
	HKAvailableWithdrawal Decimal `json:"hk_avl_withdrawal_cash"`
	HKNetCashPower        Decimal `json:"hk_net_cash_power"`
	USCash                Decimal `json:"us_cash"`
	USAvailableWithdrawal Decimal `json:"us_avl_withdrawal_cash"`
	USNetCashPower        Decimal `json:"us_net_cash_power"`
	JPCash                Decimal `json:"jp_cash"`
	JPAvailableWithdrawal Decimal `json:"jp_avl_withdrawal_cash"`
	JPNetCashPower        Decimal `json:"jp_net_cash_power"`
	CNCash                Decimal `json:"cn_cash"`
	CNAvailableWithdrawal Decimal `json:"cn_avl_withdrawal_cash"`
	CNNetCashPower        Decimal `json:"cn_net_cash_power"`
	SGCash                Decimal `json:"sg_cash"`
	SGAvailableWithdrawal Decimal `json:"sg_avl_withdrawal_cash"`
	SGNetCashPower        Decimal `json:"sg_net_cash_power"`
	KRCash                Decimal `json:"kr_cash"`
	KRAvailableWithdrawal Decimal `json:"kr_avl_withdrawal_cash"`
	KRNetCashPower        Decimal `json:"kr_net_cash_power"`
	MYCash                Decimal `json:"my_cash"`
	MYAvailableWithdrawal Decimal `json:"my_avl_withdrawal_cash"`
	MYNetCashPower        Decimal `json:"my_net_cash_power"`
	AUCash                Decimal `json:"au_cash"`
	AUAvailableWithdrawal Decimal `json:"au_avl_withdrawal_cash"`
	AUNetCashPower        Decimal `json:"au_net_cash_power"`
}

// TradingInfo represents account trading capacity for a single security.
type TradingInfo struct {
	MaxCashBuy          Decimal `json:"max_cash_buy"`
	MaxCashAndMarginBuy Decimal `json:"max_cash_and_margin_buy"`
	MaxPositionSell     Decimal `json:"max_position_sell"`
	MaxSellShort        Decimal `json:"max_sell_short"`
	MaxBuyBack          Decimal `json:"max_buy_back"`
	LongRequiredIM      Decimal `json:"long_required_im"`
	ShortRequiredIM     Decimal `json:"short_required_im"`
}

// Decimal preserves monetary precision and accepts JSON strings or numbers.
type Decimal string

func (d *Decimal) UnmarshalJSON(data []byte) error {
	value, err := decodeStringOrNumber(data)
	if err != nil {
		return err
	}
	*d = Decimal(value)
	return nil
}

// --- API response wrappers ---

type orderResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Order  `json:"data"`
}

type placeOrderResponse struct {
	Status       string `json:"s"`
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
	Data         struct {
		OrderID          string `json:"order_id"`
		NeedOrderConfirm bool   `json:"need_order_confirm"`
		ConfirmID        string `json:"confirm_id"`
	} `json:"d"`
}

type openOrdersResponse struct {
	Status       string        `json:"s"`
	ErrorCode    int           `json:"errcode"`
	ErrorMessage string        `json:"errmsg"`
	Data         orderListData `json:"d"`
}

type historyOrdersResponse = openOrdersResponse

type orderDetailsResponse struct {
	Status       string     `json:"s"`
	ErrorCode    int        `json:"errcode"`
	ErrorMessage string     `json:"errmsg"`
	Data         orderItems `json:"d"`
}

type orderListData struct {
	Orders    []Order `json:"orders"`
	PageFlag  string  `json:"page_flag"`
	Completed bool    `json:"completed"`
}

func (o *orderListData) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '[' {
		return json.Unmarshal(data, &o.Orders)
	}
	var wrapped struct {
		Orders    []Order `json:"orders"`
		PageFlag  string  `json:"page_flag"`
		Completed bool    `json:"completed"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	*o = orderListData(wrapped)
	return nil
}

type orderItems []Order

func (o *orderItems) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*o = nil
		return nil
	}
	if data[0] == '[' {
		var orders []Order
		if err := json.Unmarshal(data, &orders); err != nil {
			return err
		}
		*o = orders
		return nil
	}
	var wrapped struct {
		Orders []Order `json:"orders"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	*o = wrapped.Orders
	return nil
}

type positionResponse struct {
	Status       string        `json:"s"`
	ErrorCode    int           `json:"errcode"`
	ErrorMessage string        `json:"errmsg"`
	Data         positionItems `json:"d"`
}

type positionItems []Position

func (p *positionItems) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*p = nil
		return nil
	}
	if data[0] == '[' {
		var positions []Position
		if err := json.Unmarshal(data, &positions); err != nil {
			return err
		}
		*p = positions
		return nil
	}
	var wrapped struct {
		Positions []Position `json:"positions"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	*p = wrapped.Positions
	return nil
}

type dealListResponse struct {
	Status       string       `json:"s"`
	ErrorCode    int          `json:"errcode"`
	ErrorMessage string       `json:"errmsg"`
	Data         dealListData `json:"d"`
}

type dealListData struct {
	Deals     []Deal `json:"order_fills"`
	PageFlag  string `json:"page_flag"`
	Completed bool   `json:"completed"`
}

func (d *dealListData) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '[' {
		return json.Unmarshal(data, &d.Deals)
	}
	var wrapped struct {
		OrderFills []Deal `json:"order_fills"`
		Orders     []Deal `json:"orders"`
		Fills      []Deal `json:"fills"`
		PageFlag   string `json:"page_flag"`
		Completed  bool   `json:"completed"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	d.Deals = firstDealSlice(wrapped.OrderFills, wrapped.Orders, wrapped.Fills)
	d.PageFlag = wrapped.PageFlag
	d.Completed = wrapped.Completed
	return nil
}

type accountListResponse struct {
	Status       string `json:"s"`
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
	Data         struct {
		Accounts []Account `json:"accounts"`
	} `json:"d"`
}

type fundsResponse struct {
	Status       string `json:"s"`
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
	Data         Funds  `json:"d"`
}

type tradingInfoResponse struct {
	Status       string      `json:"s"`
	ErrorCode    int         `json:"errcode"`
	ErrorMessage string      `json:"errmsg"`
	Data         TradingInfo `json:"d"`
}

// --- Service methods ---

// PlaceOrder submits a new order and returns the resulting order record.
func (s *TradeService) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*Order, error) {
	payload := newPlaceOrderPayload(req)
	opts := client.CallOpts{
		Body:       payload,
		PathParams: map[string]string{"acc_id": req.Account},
	}

	var resp placeOrderResponse
	if err := s.client.Call(ctx, endpointOrderPlace, opts, &resp); err != nil {
		return nil, fmt.Errorf("place order: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	if err := requireOrderConfirmation(req.Account, resp, ""); err != nil {
		return nil, err
	}
	return newPlacedOrder(req, resp.Data.OrderID), nil
}

// ModifyOrder changes the price and/or quantity of an existing order.
func (s *TradeService) ModifyOrder(ctx context.Context, orderID string, req ModifyOrderRequest) (*Order, error) {
	opts := client.CallOpts{
		Body: newModifyOrderPayload(req),
		PathParams: map[string]string{
			"acc_id":   req.Account,
			"order_id": orderID,
		},
	}

	var resp placeOrderResponse
	if err := s.client.Call(ctx, endpointOrderModify, opts, &resp); err != nil {
		return nil, fmt.Errorf("modify order: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	if err := requireOrderConfirmation(req.Account, resp, req.Exchange); err != nil {
		return nil, err
	}
	return newModifiedOrder(orderID, req, resp.Data.OrderID), nil
}

// CancelOrder deletes an existing order.
func (s *TradeService) CancelOrder(ctx context.Context, orderID string, req CancelOrderRequest) error {
	opts := client.CallOpts{
		PathParams: map[string]string{
			"acc_id":   req.Account,
			"order_id": orderID,
		},
		Params: url.Values{"exchange": {req.Exchange}},
	}

	var resp placeOrderResponse
	if err := s.client.Call(ctx, endpointOrderCancel, opts, &resp); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return err
	}
	return requireOrderConfirmation(req.Account, resp, req.Exchange)
}

func requireOrderConfirmation(account string, resp placeOrderResponse, exchange string) error {
	if !resp.Data.NeedOrderConfirm {
		return nil
	}
	if len(resp.Data.ConfirmID) == 0 {
		return fmt.Errorf("order confirmation required but confirm_id is empty")
	}
	return &OrderConfirmationRequiredError{
		Account:   account,
		ConfirmID: resp.Data.ConfirmID,
		Exchange:  exchange,
	}
}

func (s *TradeService) ConfirmOrder(ctx context.Context, req OrderConfirmRequest) (string, error) {
	opts := client.CallOpts{
		Body:       newOrderConfirmPayload(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}

	var resp placeOrderResponse
	if err := s.client.Call(ctx, endpointOrderConfirm, opts, &resp); err != nil {
		return "", fmt.Errorf("confirm order: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return "", err
	}
	return resp.Data.OrderID, nil
}

// ListOpenOrders returns open (incomplete) orders for the given account,
// automatically paginating until all results are fetched.
func (s *TradeService) ListOpenOrders(ctx context.Context, req OpenOrdersRequest) ([]Order, error) {
	var allOrders []Order
	pageFlag := ""
	for {
		params := newOpenOrdersParams(req, pageFlag)
		opts := client.CallOpts{
			PathParams: map[string]string{"acc_id": req.Account},
			Params:     params,
		}
		var resp openOrdersResponse
		if err := s.client.Call(ctx, endpointOrderOpen, opts, &resp); err != nil {
			return nil, fmt.Errorf("list open orders: %w", err)
		}
		if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp.Data.Orders...)
		if resp.Data.Completed || len(resp.Data.PageFlag) == 0 {
			break
		}
		pageFlag = resp.Data.PageFlag
	}
	return allOrders, nil
}

// ListHistoryOrders returns historical orders for the given account,
// automatically paginating until all results are fetched.
func (s *TradeService) ListHistoryOrders(ctx context.Context, req HistoryOrdersRequest) ([]Order, error) {
	var allOrders []Order
	pageFlag := ""
	for {
		params := newHistoryOrdersParams(req, pageFlag)
		opts := client.CallOpts{
			PathParams: map[string]string{"acc_id": req.Account},
			Params:     params,
		}
		var resp historyOrdersResponse
		if err := s.client.Call(ctx, endpointOrderHistory, opts, &resp); err != nil {
			return nil, fmt.Errorf("list history orders: %w", err)
		}
		if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp.Data.Orders...)
		if resp.Data.Completed || len(resp.Data.PageFlag) == 0 {
			break
		}
		pageFlag = resp.Data.PageFlag
	}
	return allOrders, nil
}

// GetOrderDetails returns detailed order records for the given order IDs.
func (s *TradeService) GetOrderDetails(ctx context.Context, req OrderDetailsRequest) ([]Order, error) {
	opts := client.CallOpts{
		Body:       newOrderDetailsPayload(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	var resp orderDetailsResponse
	if err := s.client.Call(ctx, endpointOrderDetail, opts, &resp); err != nil {
		return nil, fmt.Errorf("get order details: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	return []Order(resp.Data), nil
}

// GetPositions returns positions for the given account, optionally filtered.
func (s *TradeService) GetPositions(ctx context.Context, account string, filter PositionFilter) ([]Position, error) {
	params := url.Values{}
	if len(filter.Code) > 0 {
		params.Set("code", filter.Code)
	}
	if len(filter.PLRatioMin) > 0 {
		params.Set("pl_ratio_min", filter.PLRatioMin)
	}
	if len(filter.PLRatioMax) > 0 {
		params.Set("pl_ratio_max", filter.PLRatioMax)
	}

	opts := client.CallOpts{
		PathParams: map[string]string{"acc_id": account},
		Params:     params,
	}
	var resp positionResponse
	if err := s.client.Call(ctx, endpointPosition, opts, &resp); err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	return []Position(resp.Data), nil
}

// ListDeals returns today's fills for the given account using default filters.
func (s *TradeService) ListDeals(ctx context.Context, account string) ([]Deal, error) {
	return s.ListTodayDeals(ctx, TodayDealsRequest{Account: account})
}

// ListTodayDeals returns today's fills for the given account,
// automatically paginating until all results are fetched.
func (s *TradeService) ListTodayDeals(ctx context.Context, req TodayDealsRequest) ([]Deal, error) {
	var allDeals []Deal
	pageFlag := ""
	for {
		params := newPagedMarketParams(req.Market, req.PageSize, pageFlag)
		opts := client.CallOpts{
			PathParams: map[string]string{"acc_id": req.Account},
			Params:     params,
		}
		var resp dealListResponse
		if err := s.client.Call(ctx, endpointDealToday, opts, &resp); err != nil {
			return nil, fmt.Errorf("list today deals: %w", err)
		}
		if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
			return nil, err
		}
		allDeals = append(allDeals, resp.Data.Deals...)
		if resp.Data.Completed || len(resp.Data.PageFlag) == 0 {
			break
		}
		pageFlag = resp.Data.PageFlag
	}
	return allDeals, nil
}

// ListHistoryDeals returns historical fills for the given account,
// automatically paginating until all results are fetched.
func (s *TradeService) ListHistoryDeals(ctx context.Context, req HistoryDealsRequest) ([]Deal, error) {
	var allDeals []Deal
	pageFlag := ""
	for {
		params := newHistoryDealsParams(req, pageFlag)
		opts := client.CallOpts{
			PathParams: map[string]string{"acc_id": req.Account},
			Params:     params,
		}
		var resp dealListResponse
		if err := s.client.Call(ctx, endpointDealHistory, opts, &resp); err != nil {
			return nil, fmt.Errorf("list history deals: %w", err)
		}
		if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
			return nil, err
		}
		allDeals = append(allDeals, resp.Data.Deals...)
		if resp.Data.Completed || len(resp.Data.PageFlag) == 0 {
			break
		}
		pageFlag = resp.Data.PageFlag
	}
	return allDeals, nil
}

// ListAccounts returns all trading accounts accessible to the current user.
func (s *TradeService) ListAccounts(ctx context.Context) ([]Account, error) {
	var resp accountListResponse
	if err := s.client.Call(ctx, endpointAccountList, client.CallOpts{}, &resp); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	return resp.Data.Accounts, nil
}

// GetFunds returns the monetary summary for the given account.
func (s *TradeService) GetFunds(ctx context.Context, account, currency string) (*Funds, error) {
	params := url.Values{"currency": {currency}}
	opts := client.CallOpts{PathParams: map[string]string{"acc_id": account}, Params: params}
	var resp fundsResponse
	if err := s.client.Call(ctx, endpointAccountFund, opts, &resp); err != nil {
		return nil, fmt.Errorf("get funds: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetTradingInfo returns maximum tradable quantities for a security.
func (s *TradeService) GetTradingInfo(ctx context.Context, req TradingInfoRequest) (*TradingInfo, error) {
	opts := client.CallOpts{
		PathParams: map[string]string{"acc_id": req.Account},
		Params:     newTradingInfoParams(req),
	}
	var resp tradingInfoResponse
	if err := s.client.Call(ctx, endpointAccountInfo, opts, &resp); err != nil {
		return nil, fmt.Errorf("get trading info: %w", err)
	}
	if err := checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func checkAccountAPIStatus(status string, code int, message string) error {
	if status == accountAPIStatusOK {
		return nil
	}
	return fmt.Errorf("account API error %d: %s", code, message)
}

// --- Helpers ---

// accountParams builds a url.Values containing the account parameter.
func accountParams(account string) url.Values {
	params := url.Values{}
	params.Set("account", account)
	return params
}

func newPlaceOrderPayload(req PlaceOrderRequest) placeOrderPayload {
	payload := placeOrderPayload{
		Code: req.Code, Side: req.Side, OrderType: req.OrderType,
		Qty: strconv.FormatInt(req.Qty, qtyFormatBase), Price: req.Price,
		TimeInForce: req.TimeInForce, LotType: req.LotType, Remark: req.Remark,
		Session: req.Session, AuxPrice: req.AuxPrice, OrderClass: req.OrderClass,
		MultiLegInfo: req.MultiLegInfo,
	}
	return payload
}

func newModifyOrderPayload(req ModifyOrderRequest) modifyOrderPayload {
	payload := modifyOrderPayload{
		Exchange: req.Exchange,
		Qty:      strconv.FormatInt(req.Qty, qtyFormatBase),
	}
	payload.Price = req.Price
	payload.AuxPrice = req.AuxPrice
	return payload
}

func newOrderConfirmPayload(req OrderConfirmRequest) orderConfirmPayload {
	return orderConfirmPayload{
		ConfirmID: req.ConfirmID,
		Exchange:  req.Exchange,
	}
}

func newOrderDetailsPayload(req OrderDetailsRequest) orderDetailsPayload {
	return orderDetailsPayload{
		OrderIDs: req.OrderIDs,
		Exchange: strings.ToUpper(strings.TrimSpace(req.Exchange)),
	}
}

func newOpenOrdersParams(req OpenOrdersRequest, pageFlag string) url.Values {
	return newPagedMarketParams(req.Market, req.PageSize, pageFlag)
}

func newPagedMarketParams(market string, pageSize int, pageFlag string) url.Values {
	params := url.Values{
		"trd_market": {normalizeTradingMarket(market)},
	}
	if len(pageFlag) > 0 {
		params.Set("page_flag", pageFlag)
	}
	if pageSize >= minPageSize && pageSize <= maxPageSize {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
	return params
}

func newHistoryOrdersParams(req HistoryOrdersRequest, pageFlag string) url.Values {
	return newHistoryQueryParams(req.Market, req.Start, req.End, req.Symbol, req.PageSize, pageFlag)
}

func newHistoryDealsParams(req HistoryDealsRequest, pageFlag string) url.Values {
	return newHistoryQueryParams(req.Market, req.Start, req.End, req.Symbol, req.PageSize, pageFlag)
}

func newHistoryQueryParams(market string, start int64, end int64, symbol string, pageSize int, pageFlag string) url.Values {
	params := url.Values{
		"trd_market": {normalizeTradingMarket(market)},
	}
	addInt64Param(params, "start", start)
	addInt64Param(params, "end", end)
	if len(strings.TrimSpace(symbol)) > 0 {
		params.Set("code", strings.TrimSpace(symbol))
	}
	if len(pageFlag) > 0 {
		params.Set("page_flag", pageFlag)
	}
	if pageSize >= minPageSize && pageSize <= maxPageSize {
		params.Set("page_size", strconv.Itoa(pageSize))
	}
	return params
}

func addInt64Param(params url.Values, key string, value int64) {
	if value > 0 {
		params.Set(key, strconv.FormatInt(value, qtyFormatBase))
	}
}

func normalizeTradingMarket(market string) string {
	value := strings.ToUpper(strings.TrimSpace(market))
	if len(value) == 0 {
		return DefaultTradingMarket
	}
	return value
}

func newTradingInfoParams(req TradingInfoRequest) url.Values {
	params := url.Values{
		"code":       {req.Code},
		"order_type": {req.OrderType},
	}
	if len(strings.TrimSpace(req.Price)) > 0 {
		params.Set("price", strings.TrimSpace(req.Price))
	}
	if len(req.OrderID) > 0 {
		params.Set("order_id", req.OrderID)
	}
	return params
}

func newPlacedOrder(req PlaceOrderRequest, orderID string) *Order {
	price, _ := strconv.ParseFloat(req.Price, bitSize64)
	return &Order{
		OrderID: orderID,
		Symbol:  req.Code,
		Side:    req.Side,
		Type:    req.OrderType,
		Price:   price,
		Qty:     req.Qty,
	}
}

func newModifiedOrder(orderID string, req ModifyOrderRequest, responseOrderID string) *Order {
	if len(responseOrderID) > 0 {
		orderID = responseOrderID
	}
	price, _ := strconv.ParseFloat(req.Price, bitSize64)
	return &Order{
		OrderID: orderID,
		Type:    orderTypeModified,
		Price:   price,
		Qty:     req.Qty,
	}
}
