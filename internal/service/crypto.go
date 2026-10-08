package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

const (
	endpointCryptoTotalBalance = "crypto.account.total_balance"
	endpointCryptoMaxQty       = "crypto.account.max_qty"
	endpointCryptoOrderPlace   = "crypto.order.place"
	endpointCryptoOrderModify  = "crypto.order.modify"
	endpointCryptoOrderCancel  = "crypto.order.cancel"
	endpointCryptoOrderActive  = "crypto.order.active"
	endpointCryptoOrderHistory = "crypto.order.history"
	endpointCryptoOrderDetail  = "crypto.order.detail"
	endpointCryptoDealFills    = "crypto.deal.fills"
	endpointCryptoDealHistory  = "crypto.deal.history"
	DefaultCryptoPageSize      = 50
)

// CryptoService provides access to cryptocurrency trading endpoints.
type CryptoService struct {
	client *client.Client
}

// NewCryptoService creates a new CryptoService backed by the given client.
func NewCryptoService(c *client.Client) *CryptoService {
	return &CryptoService{client: c}
}

// CryptoBalanceItem stores one raw cash or position row returned by the API.
type CryptoBalanceItem map[string]any

// CryptoTotalBalance represents total crypto account assets.
type CryptoTotalBalance struct {
	CashList  []CryptoBalanceItem `json:"cash_list"`
	Positions []CryptoBalanceItem `json:"positions"`
}

// CryptoMaxQtyRequest is the query for crypto purchasing power.
type CryptoMaxQtyRequest struct {
	Account   string
	OrderType string
	Currency  string
	Coin      string
	Price     string
}

// CryptoMaxQty represents maximum tradable quantities and amounts.
type CryptoMaxQty struct {
	MaxCashBuyQty    string `json:"max_cash_buy_qty"`
	MaxSellQty       string `json:"max_sell_qty"`
	MaxCashBuyAmount string `json:"max_cash_buy_amount"`
}

// CryptoOrdersRequest is the query for crypto active orders.
type CryptoOrdersRequest struct {
	Account     string
	Start       int64
	End         int64
	PageSize    int
	PageFlag    string
	Symbol      string
	OrderStatus string
	Currency    string
	Side        string
	OrderType   string
}

// CryptoOrder stores one raw crypto order row returned by the API.
type CryptoOrder map[string]any

// CryptoOrderList represents a paginated crypto order response.
type CryptoOrderList struct {
	Orders    []CryptoOrder `json:"orders"`
	PageFlag  string        `json:"page_flag"`
	Completed bool          `json:"completed"`
}

// CryptoFillsRequest is the query for crypto fills by order ID.
type CryptoFillsRequest struct {
	Account  string
	OrderID  string
	Start    int64
	End      int64
	Symbol   string
	PageSize int
	PageFlag string
}

// CryptoFill stores one raw crypto fill row returned by the API.
type CryptoFill map[string]any

// CryptoFillList represents a paginated crypto fill response.
type CryptoFillList struct {
	OrderFills []CryptoFill `json:"order_fills"`
	Fills      []CryptoFill `json:"fills"`
	Deals      []CryptoFill `json:"deals"`
	PageFlag   string       `json:"page_flag"`
	Completed  bool         `json:"completed"`
}

// CryptoPlaceOrderRequest is the payload for placing a crypto order.
type CryptoPlaceOrderRequest struct {
	Account         string
	ReqID           string
	Side            string
	Symbol          string
	OrderType       string
	TimeInForce     string
	Price           string
	Qty             string
	CashOrderQty    string
	ExpireTime      string
	ConditionalInfo map[string]any
}

type cryptoPlaceOrderPayload struct {
	ReqID           string         `json:"req_id"`
	Side            string         `json:"side"`
	Symbol          string         `json:"symbol"`
	OrderType       string         `json:"ord_type"`
	TimeInForce     string         `json:"time_in_force"`
	Price           string         `json:"price,omitempty"`
	Qty             string         `json:"qty,omitempty"`
	CashOrderQty    string         `json:"cash_order_qty,omitempty"`
	ExpireTime      string         `json:"expire_time,omitempty"`
	ConditionalInfo map[string]any `json:"conditional_info,omitempty"`
}

// CryptoModifyOrderRequest is the payload for modifying a crypto order.
type CryptoModifyOrderRequest struct {
	Account         string
	ReqID           string
	Price           string
	Qty             string
	OrderVersion    int64
	ConditionalInfo map[string]any
}

type cryptoModifyOrderPayload struct {
	ReqID           string         `json:"req_id"`
	Price           string         `json:"price"`
	Qty             string         `json:"qty"`
	OrderVersion    int64          `json:"order_version"`
	ConditionalInfo map[string]any `json:"conditional_info"`
}

// CryptoCancelOrderRequest is the payload for cancelling a crypto order.
type CryptoCancelOrderRequest struct {
	Account string
	ReqID   string
}

// CryptoOrderResult is returned by crypto order mutation endpoints.
type CryptoOrderResult struct {
	OrderID string `json:"order_id"`
}

type cryptoTotalBalanceResponse struct {
	Status       string             `json:"s"`
	ErrorCode    int                `json:"errcode"`
	ErrorMessage string             `json:"errmsg"`
	Code         int                `json:"code"`
	Message      string             `json:"message"`
	Msg          string             `json:"msg"`
	Data         CryptoTotalBalance `json:"data"`
	LegacyData   CryptoTotalBalance `json:"d"`
}

type cryptoMaxQtyResponse struct {
	Status       string       `json:"s"`
	ErrorCode    int          `json:"errcode"`
	ErrorMessage string       `json:"errmsg"`
	Code         int          `json:"code"`
	Message      string       `json:"message"`
	Msg          string       `json:"msg"`
	Data         CryptoMaxQty `json:"data"`
	LegacyData   CryptoMaxQty `json:"d"`
}

type cryptoOrderResponse struct {
	Status       string            `json:"s"`
	ErrorCode    int               `json:"errcode"`
	ErrorMessage string            `json:"errmsg"`
	Code         int               `json:"code"`
	Message      string            `json:"message"`
	Msg          string            `json:"msg"`
	Data         CryptoOrderResult `json:"data"`
	LegacyData   CryptoOrderResult `json:"d"`
}

type cryptoOrderListResponse struct {
	Status       string          `json:"s"`
	ErrorCode    int             `json:"errcode"`
	ErrorMessage string          `json:"errmsg"`
	Code         int             `json:"code"`
	Message      string          `json:"message"`
	Msg          string          `json:"msg"`
	Data         CryptoOrderList `json:"data"`
	LegacyData   CryptoOrderList `json:"d"`
}

type cryptoFillListResponse struct {
	Status       string         `json:"s"`
	ErrorCode    int            `json:"errcode"`
	ErrorMessage string         `json:"errmsg"`
	Code         int            `json:"code"`
	Message      string         `json:"message"`
	Msg          string         `json:"msg"`
	Data         CryptoFillList `json:"data"`
	LegacyData   CryptoFillList `json:"d"`
}

// GetTotalBalance returns account total assets, including cash and digital assets.
func (s *CryptoService) GetTotalBalance(ctx context.Context, account string) (*CryptoTotalBalance, error) {
	var resp cryptoTotalBalanceResponse
	opts := client.CallOpts{PathParams: map[string]string{"acc_id": account}}
	if err := s.client.Call(ctx, endpointCryptoTotalBalance, opts, &resp); err != nil {
		return nil, fmt.Errorf("get crypto total balance: %w", err)
	}
	if err := checkCryptoAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoTotalBalanceData(resp), nil
}

// GetMaxQty returns crypto maximum buy/sell quantity and cash buy amount.
func (s *CryptoService) GetMaxQty(ctx context.Context, req CryptoMaxQtyRequest) (*CryptoMaxQty, error) {
	var resp cryptoMaxQtyResponse
	opts := client.CallOpts{
		Params:     newCryptoMaxQtyParams(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoMaxQty, opts, &resp); err != nil {
		return nil, fmt.Errorf("get crypto max quantity: %w", err)
	}
	if err := checkCryptoMaxQtyAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoMaxQtyData(resp), nil
}

// ListActiveOrders returns current active crypto orders.
func (s *CryptoService) ListActiveOrders(ctx context.Context, req CryptoOrdersRequest) (*CryptoOrderList, error) {
	var resp cryptoOrderListResponse
	opts := client.CallOpts{
		Params:     newCryptoOrdersParams(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderActive, opts, &resp); err != nil {
		return nil, fmt.Errorf("list active crypto orders: %w", err)
	}
	if err := checkCryptoOrderListAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderListData(resp), nil
}

// ListHistoryOrders returns historical crypto orders.
func (s *CryptoService) ListHistoryOrders(ctx context.Context, req CryptoOrdersRequest) (*CryptoOrderList, error) {
	var resp cryptoOrderListResponse
	opts := client.CallOpts{
		Params:     newCryptoHistoryOrdersParams(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderHistory, opts, &resp); err != nil {
		return nil, fmt.Errorf("list history crypto orders: %w", err)
	}
	if err := checkCryptoOrderListAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderListData(resp), nil
}

// GetOrderDetails returns details for one or more crypto orders.
func (s *CryptoService) GetOrderDetails(ctx context.Context, account string, orderIDs []string) ([]CryptoOrder, error) {
	var resp cryptoOrderListResponse
	opts := client.CallOpts{
		PathParams: map[string]string{
			"acc_id":   account,
			"order_id": strings.Join(orderIDs, ","),
		},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderDetail, opts, &resp); err != nil {
		return nil, fmt.Errorf("get crypto order details: %w", err)
	}
	if err := checkCryptoOrderListAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderListData(resp).Orders, nil
}

// ListFills returns crypto fill records for an order.
func (s *CryptoService) ListFills(ctx context.Context, req CryptoFillsRequest) (*CryptoFillList, error) {
	var resp cryptoFillListResponse
	opts := client.CallOpts{
		Params:     newCryptoFillsParams(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoDealFills, opts, &resp); err != nil {
		return nil, fmt.Errorf("list crypto fills: %w", err)
	}
	if err := checkCryptoFillListAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoFillListData(resp), nil
}

// ListHistoryFills returns historical crypto fill records.
func (s *CryptoService) ListHistoryFills(ctx context.Context, req CryptoFillsRequest) (*CryptoFillList, error) {
	var resp cryptoFillListResponse
	opts := client.CallOpts{
		Params:     newCryptoHistoryFillsParams(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoDealHistory, opts, &resp); err != nil {
		return nil, fmt.Errorf("list history crypto fills: %w", err)
	}
	if err := checkCryptoFillListAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoFillListData(resp), nil
}

// PlaceOrder submits a new crypto order and returns the resulting order ID.
func (s *CryptoService) PlaceOrder(ctx context.Context, req CryptoPlaceOrderRequest) (*CryptoOrderResult, error) {
	var resp cryptoOrderResponse
	opts := client.CallOpts{
		Body:       newCryptoPlaceOrderPayload(req),
		PathParams: map[string]string{"acc_id": req.Account},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderPlace, opts, &resp); err != nil {
		return nil, fmt.Errorf("place crypto order: %w", err)
	}
	if err := checkCryptoOrderAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderResult(resp), nil
}

// ModifyOrder changes the price, quantity, or condition of a crypto order.
func (s *CryptoService) ModifyOrder(
	ctx context.Context, orderID string, req CryptoModifyOrderRequest,
) (*CryptoOrderResult, error) {
	var resp cryptoOrderResponse
	opts := client.CallOpts{
		Body: newCryptoModifyOrderPayload(req),
		PathParams: map[string]string{
			"acc_id":   req.Account,
			"order_id": orderID,
		},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderModify, opts, &resp); err != nil {
		return nil, fmt.Errorf("modify crypto order: %w", err)
	}
	if err := checkCryptoOrderAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderResult(resp), nil
}

// CancelOrder cancels a crypto order.
func (s *CryptoService) CancelOrder(
	ctx context.Context, orderID string, req CryptoCancelOrderRequest,
) (*CryptoOrderResult, error) {
	var resp cryptoOrderResponse
	opts := client.CallOpts{
		Params: newCryptoCancelOrderParams(req),
		PathParams: map[string]string{
			"acc_id":   req.Account,
			"order_id": orderID,
		},
	}
	if err := s.client.Call(ctx, endpointCryptoOrderCancel, opts, &resp); err != nil {
		return nil, fmt.Errorf("cancel crypto order: %w", err)
	}
	if err := checkCryptoOrderAPIStatus(resp); err != nil {
		return nil, err
	}
	return cryptoOrderResult(resp), nil
}

func checkCryptoAPIStatus(resp cryptoTotalBalanceResponse) error {
	if len(resp.Status) > 0 {
		return checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage)
	}
	return checkCode(resp.Code, firstString(resp.Message, resp.Msg, resp.ErrorMessage))
}

func checkCryptoMaxQtyAPIStatus(resp cryptoMaxQtyResponse) error {
	if len(resp.Status) > 0 {
		return checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage)
	}
	return checkCode(resp.Code, firstString(resp.Message, resp.Msg, resp.ErrorMessage))
}

func checkCryptoOrderAPIStatus(resp cryptoOrderResponse) error {
	if len(resp.Status) > 0 {
		return checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage)
	}
	return checkCode(resp.Code, firstString(resp.Message, resp.Msg, resp.ErrorMessage))
}

func checkCryptoOrderListAPIStatus(resp cryptoOrderListResponse) error {
	if len(resp.Status) > 0 {
		return checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage)
	}
	return checkCode(resp.Code, firstString(resp.Message, resp.Msg, resp.ErrorMessage))
}

func checkCryptoFillListAPIStatus(resp cryptoFillListResponse) error {
	if len(resp.Status) > 0 {
		return checkAccountAPIStatus(resp.Status, resp.ErrorCode, resp.ErrorMessage)
	}
	return checkCode(resp.Code, firstString(resp.Message, resp.Msg, resp.ErrorMessage))
}

func cryptoTotalBalanceData(resp cryptoTotalBalanceResponse) *CryptoTotalBalance {
	if len(resp.Data.CashList) > 0 || len(resp.Data.Positions) > 0 {
		return &resp.Data
	}
	return &resp.LegacyData
}

func cryptoMaxQtyData(resp cryptoMaxQtyResponse) *CryptoMaxQty {
	if len(resp.Data.MaxCashBuyQty) > 0 || len(resp.Data.MaxSellQty) > 0 || len(resp.Data.MaxCashBuyAmount) > 0 {
		return &resp.Data
	}
	return &resp.LegacyData
}

func cryptoOrderListData(resp cryptoOrderListResponse) *CryptoOrderList {
	if len(resp.Data.Orders) > 0 || len(resp.Data.PageFlag) > 0 || resp.Data.Completed {
		return &resp.Data
	}
	return &resp.LegacyData
}

func cryptoFillListData(resp cryptoFillListResponse) *CryptoFillList {
	data := normalizeCryptoFillList(resp.Data)
	if len(data.OrderFills) > 0 || len(data.PageFlag) > 0 || data.Completed {
		return &data
	}
	legacyData := normalizeCryptoFillList(resp.LegacyData)
	return &legacyData
}

func normalizeCryptoFillList(list CryptoFillList) CryptoFillList {
	if len(list.OrderFills) > 0 {
		return list
	}
	if len(list.Fills) > 0 {
		list.OrderFills = list.Fills
		return list
	}
	if len(list.Deals) > 0 {
		list.OrderFills = list.Deals
		return list
	}
	return list
}

func cryptoOrderResult(resp cryptoOrderResponse) *CryptoOrderResult {
	if len(resp.Data.OrderID) > 0 {
		return &resp.Data
	}
	return &resp.LegacyData
}

func newCryptoPlaceOrderPayload(req CryptoPlaceOrderRequest) cryptoPlaceOrderPayload {
	return cryptoPlaceOrderPayload{
		ReqID:           req.ReqID,
		Side:            req.Side,
		Symbol:          req.Symbol,
		OrderType:       req.OrderType,
		TimeInForce:     req.TimeInForce,
		Price:           req.Price,
		Qty:             req.Qty,
		CashOrderQty:    req.CashOrderQty,
		ExpireTime:      req.ExpireTime,
		ConditionalInfo: req.ConditionalInfo,
	}
}

func newCryptoMaxQtyParams(req CryptoMaxQtyRequest) url.Values {
	params := url.Values{}
	addQueryParam(params, "order_type", req.OrderType)
	addQueryParam(params, "currency", req.Currency)
	addQueryParam(params, "coin", req.Coin)
	addQueryParam(params, "price", req.Price)
	return params
}

func newCryptoOrdersParams(req CryptoOrdersRequest) url.Values {
	params := url.Values{}
	if req.PageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", req.PageSize))
	}
	addQueryParam(params, "page_flag", req.PageFlag)
	return params
}

func newCryptoHistoryOrdersParams(req CryptoOrdersRequest) url.Values {
	params := newCryptoOrdersParams(req)
	if req.Start > 0 {
		params.Set("start_time", fmt.Sprintf("%d", req.Start))
	}
	if req.End > 0 {
		params.Set("end_time", fmt.Sprintf("%d", req.End))
	}
	addQueryParam(params, "symbol", req.Symbol)
	addQueryParam(params, "order_status", req.OrderStatus)
	addQueryParam(params, "currency", req.Currency)
	addQueryParam(params, "side", req.Side)
	addQueryParam(params, "ord_type", req.OrderType)
	return params
}

func newCryptoFillsParams(req CryptoFillsRequest) url.Values {
	params := url.Values{}
	addQueryParam(params, "order_id", req.OrderID)
	if req.PageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", req.PageSize))
	}
	addQueryParam(params, "page_flag", req.PageFlag)
	return params
}

func newCryptoHistoryFillsParams(req CryptoFillsRequest) url.Values {
	params := url.Values{}
	if req.Start > 0 {
		params.Set("start_time", fmt.Sprintf("%d", req.Start))
	}
	if req.End > 0 {
		params.Set("end_time", fmt.Sprintf("%d", req.End))
	}
	if req.PageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", req.PageSize))
	}
	addQueryParam(params, "page_flag", req.PageFlag)
	addQueryParam(params, "symbol", req.Symbol)
	return params
}

func addQueryParam(params url.Values, key string, value string) {
	if len(value) > 0 {
		params.Set(key, value)
	}
}

func newCryptoModifyOrderPayload(req CryptoModifyOrderRequest) cryptoModifyOrderPayload {
	return cryptoModifyOrderPayload{
		ReqID:           req.ReqID,
		Price:           req.Price,
		Qty:             req.Qty,
		OrderVersion:    req.OrderVersion,
		ConditionalInfo: req.ConditionalInfo,
	}
}

func newCryptoCancelOrderParams(req CryptoCancelOrderRequest) url.Values {
	params := url.Values{}
	addQueryParam(params, "req_id", req.ReqID)
	return params
}

func DecodeConditionalInfo(value string) (map[string]any, error) {
	if len(value) == 0 {
		return nil, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, fmt.Errorf("decode conditional info: %w", err)
	}
	return decoded, nil
}
