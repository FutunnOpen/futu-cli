package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
	"github.com/FutunnOpen/futu-cli/pkg/symbol"
)

// Order side constants.
const (
	orderSideBuy       = "BUY"
	orderSideSell      = "SELL"
	orderSideSellShort = "SELL_SHORT"
	orderSideBuyBack   = "BUY_BACK"
)

// Order type constants.
const (
	orderTypeLimit       = "LIMIT"
	orderTypeMarket      = "MARKET"
	orderTypeInputMarket = "market"
	defaultOrderType     = orderTypeInputMarket
	defaultTimeInForce   = "DAY"
	orderTypeModified    = "MODIFIED"
	orderClassMultiLeg   = "MLEG"
)

// Price format precision for order display.
const orderPriceFormat = "%.3f"

// Quantity argument position index.
const qtyArgIndex = 1

// Integer parsing base and bit size for quantity.
const (
	qtyParseBase    = 10
	qtyParseBitSize = 64
	maxOrderIDs     = 100
)

// Table header labels for the order list command.
const (
	orderHeaderID     = "OrderID"
	orderHeaderSymbol = "Symbol"
	orderHeaderSide   = "Side"
	orderHeaderType   = "Type"
	orderHeaderPrice  = "Price"
	orderHeaderQty    = "Qty"
	orderHeaderFilled = "Filled"
	orderHeaderStatus = "Status"
	orderHeaderTime   = "Time"
)

// Table header labels for the maximum quantity command.
const (
	tradingInfoHeaderKey             = "Key"
	tradingInfoHeaderValue           = "Value"
	tradingInfoLabelMaxCashBuy       = "Max Cash Buy"
	tradingInfoLabelMaxCashMarginBuy = "Max Cash + Margin Buy"
	tradingInfoLabelMaxPositionSell  = "Max Position Sell"
	tradingInfoLabelMaxSellShort     = "Max Sell Short"
	tradingInfoLabelMaxBuyBack       = "Max Buy Back"
	tradingInfoLabelLongRequiredIM   = "Long Required IM"
	tradingInfoLabelShortRequiredIM  = "Short Required IM"
)

// Table header labels for detailed order output.
const (
	orderDetailHeaderField = "Field"
	orderDetailHeaderValue = "Value"
)

var orderSideLabels = map[string]string{
	"BUY":        "买入",
	"SELL":       "卖出",
	"SELL_SHORT": "卖空",
	"BUY_BACK":   "买回",
}

var orderTypeLabels = map[string]string{
	"LIMIT":             "限价单",
	"MARKET":            "市价单",
	"AUCTION":           "竞价市价单",
	"AUCTION_LIMIT":     "竞价限价单",
	"STOP":              "止损单",
	"STOP_LIMIT":        "止损限价单",
	"MARKET_IF_TOUCHED": "触及市价单",
	"LIMIT_IF_TOUCHED":  "触及限价单",
	orderTypeModified:   "已改单",
}

var orderStatusLabels = map[string]string{
	"WAITING_SUBMIT": "等待提交",
	"SUBMITTING":     "提交中",
	"SUBMITTED":      "已提交",
	"FILLED_PART":    "部分成交",
	"FILLED_ALL":     "全部成交",
	"CANCELLED_PART": "部分成交剩余已撤",
	"CANCELLED_ALL":  "全部撤单",
	"FAILED":         "下单失败",
	"DISABLED":       "已失效",
	"DELETED":        "已删除",
}

// orderFlags holds shared flag values for order subcommands.
var orderFlags struct {
	price        string
	auxPrice     string
	qty          int64
	typ          string
	account      string
	exchange     string
	orderID      string
	market       string
	pageSize     int
	start        int64
	end          int64
	symbol       string
	side         string
	timeInForce  string
	lotType      string
	remark       string
	session      string
	orderClass   string
	multiLegInfo string
}

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "交易订单管理",
	Long:  "管理交易订单：下单买入/卖出、撤单、查看订单列表。",
}

var orderBuyCmd = &cobra.Command{
	Use:   "buy <symbol> <qty>",
	Short: "买入下单",
	Long:  "提交买入订单，支持限价单和市价单。",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaceOrder(cmd, args, orderSideBuy)
	},
}

var orderSellCmd = &cobra.Command{
	Use:   "sell <symbol> <qty>",
	Short: "卖出下单",
	Long:  "提交卖出订单，支持限价单和市价单。",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaceOrder(cmd, args, orderSideSell)
	},
}

var orderPlaceCmd = &cobra.Command{
	Use:   "place <qty>",
	Short: "提交通用证券订单",
	Long:  "提交支持卖空、买回、条件单和多腿订单的证券订单。普通订单需通过 --symbol 指定标的。",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenericPlaceOrder,
}

var orderCancelCmd = &cobra.Command{
	Use:   "cancel <order-id>",
	Short: "撤销订单",
	Long:  "撤销指定的挂单。",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancelOrder,
}

var orderModifyCmd = &cobra.Command{
	Use:   "modify <order-id>",
	Short: "修改订单",
	Long:  "修改指定订单的数量、价格或条件单触发价。",
	Args:  cobra.ExactArgs(1),
	RunE:  runModifyOrder,
}

var orderListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看未完成订单",
	Long:  "获取当前账户的未完成订单列表，包含最近 24 小时内已成交或已撤销的订单。",
	Args:  cobra.NoArgs,
	RunE:  runListOrders,
}

var orderHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "查看历史订单",
	Long:  "获取当前账户的历史订单列表，支持按市场、时间范围和标的过滤。",
	Args:  cobra.NoArgs,
	RunE:  runHistoryOrders,
}

var orderDetailCmd = &cobra.Command{
	Use:   "detail <order-id> [order-id...]",
	Short: "查看订单详情",
	Long:  "批量获取指定订单 ID 的详情，最多一次查询 100 个订单。",
	Args:  cobra.RangeArgs(1, maxOrderIDs),
	RunE:  runOrderDetails,
}

var orderMaxQtyCmd = &cobra.Command{
	Use:   "max-qty <symbol>",
	Short: "查询最大买卖数量",
	Long:  "查询指定标的的最大现金买入、融资买入、持仓卖出、卖空和回补买入数量。",
	Args:  cobra.ExactArgs(1),
	RunE:  runMaxOrderQty,
}

func init() {
	registerBuyFlags(orderBuyCmd)
	registerSellFlags(orderSellCmd)
	registerGenericPlaceFlags(orderPlaceCmd)
	registerModifyFlags(orderModifyCmd)
	registerCancelFlags(orderCancelCmd)
	registerListOrderFlags(orderListCmd)
	registerHistoryOrderFlags(orderHistoryCmd)
	registerDetailOrderFlags(orderDetailCmd)
	registerMaxQtyFlags(orderMaxQtyCmd)

	orderCmd.AddCommand(orderBuyCmd, orderSellCmd, orderPlaceCmd, orderModifyCmd, orderCancelCmd, orderListCmd, orderHistoryCmd, orderDetailCmd, orderMaxQtyCmd)
	rootCmd.AddCommand(orderCmd)
}

// registerBuyFlags adds --price, --type, and --account flags to a buy command.
func registerBuyFlags(cmd *cobra.Command) {
	registerPlaceFlags(cmd)
}

// registerSellFlags adds --price, --type, and --account flags to a sell command.
func registerSellFlags(cmd *cobra.Command) {
	registerPlaceFlags(cmd)
}

func registerGenericPlaceFlags(cmd *cobra.Command) {
	registerPlaceFlags(cmd)
	cmd.Flags().StringVar(&orderFlags.symbol, "symbol", "", "标的代码（默认: 空；非多腿订单必填）")
	cmd.Flags().StringVar(&orderFlags.side, "side", "", "交易方向: BUY, SELL, SELL_SHORT, BUY_BACK（必填；默认: 空）")
	_ = cmd.MarkFlagRequired("side")
}

func registerPlaceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "委托价格字符串（默认: 空；市价单不传价格）")
	cmd.Flags().StringVarP(&orderFlags.typ, "type", "t", defaultOrderType, "订单类型（默认: market）")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	cmd.Flags().StringVar(&orderFlags.timeInForce, "time-in-force", defaultTimeInForce, "订单有效期: DAY, GTC（默认: DAY）")
	cmd.Flags().StringVar(&orderFlags.lotType, "lot-type", "", "港股手数类型: ROUND, ODD（默认: 空，由后端决定）")
	cmd.Flags().StringVar(&orderFlags.remark, "remark", "", "订单备注，最多 64 字节（默认: 空）")
	cmd.Flags().StringVar(&orderFlags.session, "session", "", "交易时段: RTH, RTH+Pre/Post-Mkt, OVERNIGHT, ALL_DAY（默认: 空，由后端决定）")
	cmd.Flags().StringVar(&orderFlags.auxPrice, "aux-price", "", "条件单触发价格字符串（默认: 空，不传）")
	cmd.Flags().StringVar(&orderFlags.orderClass, "order-class", "", "订单类别: MLEG（默认: 空，普通订单）")
	cmd.Flags().StringVar(&orderFlags.multiLegInfo, "multi-leg-info", "", "多腿订单 JSON 对象（默认: 空）")
}

// registerListOrderFlags adds flags for the list orders command.
func registerListOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	cmd.Flags().StringVarP(&orderFlags.market, "market", "m", service.DefaultTradingMarket, "交易市场: HK, US, HKCC, SG, JP, AU, MY, KR（默认: HK）")
	cmd.Flags().IntVar(&orderFlags.pageSize, "page-size", service.DefaultPageSize, "每页数量，范围 10-100（默认: 50）")
}

// registerHistoryOrderFlags adds flags for the history orders command.
func registerHistoryOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	cmd.Flags().StringVarP(&orderFlags.market, "market", "m", service.DefaultTradingMarket, "交易市场: HK, US, HKCC, SG, JP, AU, MY, KR（默认: HK）")
	cmd.Flags().Int64Var(&orderFlags.start, "start", 0, "查询开始时间，微秒时间戳（默认: 不限制）")
	cmd.Flags().Int64Var(&orderFlags.end, "end", 0, "查询结束时间，微秒时间戳（默认: 不限制）")
	cmd.Flags().StringVar(&orderFlags.symbol, "symbol", "", "标的代码过滤（默认: 不过滤）")
	cmd.Flags().IntVar(&orderFlags.pageSize, "page-size", service.DefaultPageSize, "每页数量，范围 10-100（默认: 50）")
}

// registerDetailOrderFlags adds flags for the detail order command.
func registerDetailOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "交易所标识（必填，如 US、SEHK）")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerModifyFlags adds flags required by the modify order command.
func registerModifyFlags(cmd *cobra.Command) {
	cmd.Flags().Int64Var(&orderFlags.qty, "qty", 0, "新订单数量（必填）")
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "新委托价格字符串（限价单必填；默认: 不修改）")
	cmd.Flags().StringVar(&orderFlags.auxPrice, "aux-price", "", "新触发价格字符串（条件单使用；默认: 不修改）")
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "交易所标识（必填，如 HK、US）")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	_ = cmd.MarkFlagRequired("qty")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerCancelFlags adds flags required by the cancel order command.
func registerCancelFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "交易所标识（必填，如 HK、US）")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerMaxQtyFlags adds flags for the max-qty command.
func registerMaxQtyFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "参考价格字符串（非市价单必填；默认: 空）")
	cmd.Flags().StringVarP(&orderFlags.typ, "type", "t", defaultOrderType, "订单类型: limit, market（默认: market）")
	cmd.Flags().StringVar(&orderFlags.orderID, "order-id", "", "改单时的原订单 ID（默认: 空）")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "交易账户（默认: 配置 default_account）")
}

// runPlaceOrder is the shared handler for buy and sell subcommands.
func runPlaceOrder(cmd *cobra.Command, args []string, side string) error {
	req, err := buildPlaceOrderRequest(args[0], args[qtyArgIndex], side)
	if err != nil {
		return err
	}

	printOrderSummary(req)

	if !confirmOrder() {
		fmt.Fprintln(os.Stderr, "Order cancelled.")
		return nil
	}

	return submitOrder(cmd, req)
}

// buildPlaceOrderRequest parses arguments and flags into a PlaceOrderRequest.
func runGenericPlaceOrder(cmd *cobra.Command, args []string) error {
	req, err := buildPlaceOrderRequest(orderFlags.symbol, args[0], orderFlags.side)
	if err != nil {
		return err
	}
	printOrderSummary(req)
	if !confirmOrder() {
		fmt.Fprintln(os.Stderr, "Order cancelled.")
		return nil
	}
	return submitOrder(cmd, req)
}

func buildPlaceOrderRequest(rawSymbol, rawQty, side string) (service.PlaceOrderRequest, error) {
	qty, account, err := parseOrderQuantityAndAccount(rawQty)
	if err != nil {
		return service.PlaceOrderRequest{}, err
	}
	req := service.PlaceOrderRequest{Account: account, Qty: qty}
	if err := applyPlaceOrderFlags(&req, rawSymbol, side); err != nil {
		return service.PlaceOrderRequest{}, err
	}
	return req, nil
}

func parseOrderQuantityAndAccount(rawQty string) (int64, string, error) {
	qty, err := strconv.ParseInt(rawQty, qtyParseBase, qtyParseBitSize)
	if err != nil {
		return 0, "", fmt.Errorf("parsing quantity: %w", err)
	}
	if qty <= 0 {
		return 0, "", fmt.Errorf("quantity must be greater than 0")
	}
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return 0, "", fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	return qty, account, nil
}

func applyPlaceOrderFlags(req *service.PlaceOrderRequest, rawSymbol, side string) error {
	orderSide, err := normalizeOrderSide(side)
	if err != nil {
		return err
	}
	orderType, err := normalizeOrderType(orderFlags.typ)
	if err != nil {
		return err
	}
	code, err := parsePlaceOrderSymbol(rawSymbol, orderFlags.orderClass)
	if err != nil {
		return err
	}
	multiLegInfo, err := parseMultiLegInfo(orderFlags.multiLegInfo)
	if err != nil {
		return err
	}
	price, err := normalizePriceString(orderFlags.price)
	if err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}
	auxPrice, err := normalizePriceString(orderFlags.auxPrice)
	if err != nil {
		return fmt.Errorf("invalid aux-price: %w", err)
	}
	req.Code, req.Side, req.OrderType = code, orderSide, orderType
	req.Price, req.AuxPrice = price, auxPrice
	req.TimeInForce = strings.ToUpper(strings.TrimSpace(orderFlags.timeInForce))
	req.LotType, req.Remark = strings.ToUpper(strings.TrimSpace(orderFlags.lotType)), orderFlags.remark
	req.Session = strings.TrimSpace(orderFlags.session)
	req.OrderClass, req.MultiLegInfo = strings.ToUpper(strings.TrimSpace(orderFlags.orderClass)), multiLegInfo
	return nil
}

func normalizePriceString(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return "", nil
	}
	parsed, err := strconv.ParseFloat(value, qtyParseBitSize)
	if err != nil || parsed <= 0 || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return "", fmt.Errorf("must be a positive number")
	}
	return value, nil
}

func parsePlaceOrderSymbol(rawSymbol, orderClass string) (string, error) {
	if len(strings.TrimSpace(rawSymbol)) == 0 {
		if strings.EqualFold(strings.TrimSpace(orderClass), orderClassMultiLeg) {
			return "", nil
		}
		return "", fmt.Errorf("symbol is required for non-multi-leg orders")
	}
	sym, err := symbol.Parse(rawSymbol)
	if err != nil {
		return "", fmt.Errorf("parsing symbol: %w", err)
	}
	return sym.FutuCode(), nil
}

func parseMultiLegInfo(value string) (json.RawMessage, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return nil, nil
	}
	data := json.RawMessage(value)
	var object map[string]any
	if !json.Valid(data) || json.Unmarshal(data, &object) != nil || object == nil {
		return nil, fmt.Errorf("multi-leg-info must be a JSON object")
	}
	return data, nil
}

// printOrderSummary writes a human-readable order summary to stderr.
func printOrderSummary(req service.PlaceOrderRequest) {
	fmt.Fprintf(os.Stderr, "\nOrder Summary:\n")
	fmt.Fprintf(os.Stderr, "  Side:    %s\n", req.Side)
	fmt.Fprintf(os.Stderr, "  Symbol:  %s\n", req.Code)
	fmt.Fprintf(os.Stderr, "  Qty:     %d\n", req.Qty)
	fmt.Fprintf(os.Stderr, "  Type:    %s\n", req.OrderType)
	fmt.Fprintf(os.Stderr, "  Price:   %s\n", req.Price)
	fmt.Fprintf(os.Stderr, "  Account: %s\n\n", req.Account)
}

func normalizeOrderType(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if _, ok := orderTypeLabels[normalized]; !ok || normalized == orderTypeModified {
		return "", fmt.Errorf("unsupported order type %q", value)
	}
	return normalized, nil
}

func normalizeOrderSide(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case orderSideBuy, orderSideSell, orderSideSellShort, orderSideBuyBack:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported order side %q", value)
	}
}

// confirmOrder prompts the user for confirmation and returns true if accepted.
func confirmOrder() bool {
	return confirmAction("Confirm order?")
}

func confirmAction(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s (y/n): ", prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer := strings.TrimSpace(input)
	return len(answer) > 0 && (answer[0] == 'y' || answer[0] == 'Y')
}

// submitOrder calls the trade service to place the order and prints the result.
func submitOrder(cmd *cobra.Command, req service.PlaceOrderRequest) error {
	ctx := cmdContext(cmd)
	order, err := tradeSvc.PlaceOrder(ctx, req)
	if err != nil {
		orderID, confirmErr := handleOrderConfirmation(ctx, err, "Confirm order with broker?")
		if confirmErr != nil {
			return fmt.Errorf("placing order: %w", confirmErr)
		}
		order = &service.Order{
			OrderID: orderID,
			Symbol:  req.Code,
			Side:    req.Side,
			Type:    req.OrderType,
			Price:   parseDisplayPrice(req.Price),
			Qty:     req.Qty,
		}
	}

	printResult(order, func() {
		renderSingleOrder(order)
	})
	return nil
}

// runModifyOrder handles the modify subcommand.
func runModifyOrder(cmd *cobra.Command, args []string) error {
	req, err := buildModifyOrderRequest()
	if err != nil {
		return err
	}
	printModifyOrderSummary(args[0], req)
	if !confirmAction("Confirm order modification?") {
		fmt.Fprintln(os.Stderr, "Order modification cancelled.")
		return nil
	}

	ctx := cmdContext(cmd)
	order, err := tradeSvc.ModifyOrder(ctx, args[0], req)
	if err != nil {
		if _, confirmErr := handleOrderConfirmation(ctx, err, "Confirm order modification with broker?"); confirmErr != nil {
			return fmt.Errorf("modifying order: %w", confirmErr)
		}
		order = &service.Order{
			OrderID: args[0],
			Type:    orderTypeModified,
			Price:   parseDisplayPrice(req.Price),
			Qty:     req.Qty,
		}
		printResult(order, func() {
			renderSingleOrder(order)
		})
		return nil
	}

	printResult(order, func() {
		renderSingleOrder(order)
	})
	return nil
}

func printModifyOrderSummary(orderID string, req service.ModifyOrderRequest) {
	fmt.Fprintf(os.Stderr, "\nOrder Modification Summary:\n  Order ID: %s\n  Qty:      %d\n  Price:    %s\n  AuxPrice: %s\n  Exchange: %s\n  Account:  %s\n\n", orderID, req.Qty, req.Price, req.AuxPrice, req.Exchange, req.Account)
}

func parseDisplayPrice(value string) float64 {
	price, _ := strconv.ParseFloat(value, qtyParseBitSize)
	return price
}

func handleOrderConfirmation(ctx context.Context, err error, prompt string) (string, error) {
	var required *service.OrderConfirmationRequiredError
	if !errors.As(err, &required) {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "Broker requires secondary confirmation. ConfirmID: %s\n", required.ConfirmID)
	if !confirmAction(prompt) {
		return "", fmt.Errorf("order confirmation cancelled")
	}
	orderID, err := tradeSvc.ConfirmOrder(ctx, service.OrderConfirmRequest{
		Account:   required.Account,
		ConfirmID: required.ConfirmID,
		Exchange:  required.Exchange,
	})
	return orderID, err
}

// buildModifyOrderRequest parses flags into a ModifyOrderRequest.
func buildModifyOrderRequest() (service.ModifyOrderRequest, error) {
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return service.ModifyOrderRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	if orderFlags.qty <= 0 {
		return service.ModifyOrderRequest{}, fmt.Errorf("qty must be greater than 0")
	}
	if len(strings.TrimSpace(orderFlags.exchange)) == 0 {
		return service.ModifyOrderRequest{}, fmt.Errorf("exchange is required")
	}
	price, err := normalizePriceString(orderFlags.price)
	if err != nil {
		return service.ModifyOrderRequest{}, fmt.Errorf("invalid price: %w", err)
	}
	auxPrice, err := normalizePriceString(orderFlags.auxPrice)
	if err != nil {
		return service.ModifyOrderRequest{}, fmt.Errorf("invalid aux-price: %w", err)
	}
	return service.ModifyOrderRequest{
		Account:  account,
		Exchange: strings.ToUpper(strings.TrimSpace(orderFlags.exchange)),
		Qty:      orderFlags.qty,
		Price:    price,
		AuxPrice: auxPrice,
	}, nil
}

// renderSingleOrder writes a single order as a formatted table to stdout.
func renderSingleOrder(order *service.Order) {
	headers, row := singleOrderColumns(order)
	tw := output.NewTableWriter(os.Stdout, headers...)
	tw.AddRow(row...)
	tw.Render()
}

func singleOrderColumns(order *service.Order) ([]string, []string) {
	headers := []string{orderHeaderID}
	row := []string{order.OrderID}

	addTextOrderColumn(&headers, &row, orderHeaderSymbol, order.Symbol)
	addTextOrderColumn(&headers, &row, orderHeaderSide, formatOrderValue(order.Side, orderSideLabels))
	addTextOrderColumn(&headers, &row, orderHeaderType, formatOrderValue(order.Type, orderTypeLabels))
	addPriceOrderColumn(&headers, &row, order.Price)
	addIntOrderColumn(&headers, &row, orderHeaderQty, order.Qty)
	addIntOrderColumn(&headers, &row, orderHeaderFilled, order.FilledQty)
	addTextOrderColumn(&headers, &row, orderHeaderStatus, formatOrderValue(order.Status, orderStatusLabels))
	addTextOrderColumn(&headers, &row, orderHeaderTime, formatTimestamp(order.CreateTime))

	return headers, row
}

func addTextOrderColumn(headers *[]string, row *[]string, header string, value string) {
	if len(value) == 0 || value == "-" {
		return
	}
	*headers = append(*headers, header)
	*row = append(*row, value)
}

func addPriceOrderColumn(headers *[]string, row *[]string, value float64) {
	if value <= 0 {
		return
	}
	*headers = append(*headers, orderHeaderPrice)
	*row = append(*row, fmt.Sprintf(orderPriceFormat, value))
}

func addIntOrderColumn(headers *[]string, row *[]string, header string, value int64) {
	if value <= 0 {
		return
	}
	*headers = append(*headers, header)
	*row = append(*row, fmt.Sprintf("%d", value))
}

// runCancelOrder handles the cancel subcommand.
func runCancelOrder(cmd *cobra.Command, args []string) error {
	orderID := args[0]
	req, err := buildCancelOrderRequest()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nOrder Cancellation Summary:\n  Order ID: %s\n  Exchange: %s\n  Account:  %s\n\n", orderID, req.Exchange, req.Account)
	if !confirmAction("Confirm order cancellation?") {
		fmt.Fprintln(os.Stderr, "Order cancellation cancelled.")
		return nil
	}
	ctx := cmdContext(cmd)

	if err := tradeSvc.CancelOrder(ctx, orderID, req); err != nil {
		if _, confirmErr := handleOrderConfirmation(ctx, err, "Confirm order cancellation with broker?"); confirmErr != nil {
			return fmt.Errorf("cancelling order: %w", confirmErr)
		}
		fmt.Fprintf(os.Stdout, "Order cancelled: %s\n", orderID)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Order cancelled: %s\n", orderID)
	return nil
}

// buildCancelOrderRequest parses flags into a CancelOrderRequest.
func buildCancelOrderRequest() (service.CancelOrderRequest, error) {
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return service.CancelOrderRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	if len(strings.TrimSpace(orderFlags.exchange)) == 0 {
		return service.CancelOrderRequest{}, fmt.Errorf("exchange is required")
	}
	return service.CancelOrderRequest{
		Account:  account,
		Exchange: strings.ToUpper(strings.TrimSpace(orderFlags.exchange)),
	}, nil
}

// runListOrders handles the list subcommand.
func runListOrders(cmd *cobra.Command, args []string) error {
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	ctx := cmdContext(cmd)

	req := service.OpenOrdersRequest{
		Account:  account,
		Market:   strings.TrimSpace(orderFlags.market),
		PageSize: orderFlags.pageSize,
	}
	orders, err := tradeSvc.ListOpenOrders(ctx, req)
	if err != nil {
		return fmt.Errorf("listing open orders: %w", err)
	}

	printResult(orders, func() {
		renderOrderTable(orders)
	})
	return nil
}

// runHistoryOrders handles the history subcommand.
func runHistoryOrders(cmd *cobra.Command, args []string) error {
	req, err := buildHistoryOrdersRequest()
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	orders, err := tradeSvc.ListHistoryOrders(ctx, req)
	if err != nil {
		return fmt.Errorf("listing history orders: %w", err)
	}
	printResult(orders, func() {
		renderOrderTable(orders)
	})
	return nil
}

func buildHistoryOrdersRequest() (service.HistoryOrdersRequest, error) {
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return service.HistoryOrdersRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	return service.HistoryOrdersRequest{
		Account:  account,
		Market:   strings.TrimSpace(orderFlags.market),
		Start:    orderFlags.start,
		End:      orderFlags.end,
		Symbol:   strings.TrimSpace(orderFlags.symbol),
		PageSize: orderFlags.pageSize,
	}, nil
}

// runOrderDetails handles the detail subcommand.
func runOrderDetails(cmd *cobra.Command, args []string) error {
	req, err := buildOrderDetailsRequest(args)
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	orders, err := tradeSvc.GetOrderDetails(ctx, req)
	if err != nil {
		return fmt.Errorf("getting order details: %w", err)
	}
	printResult(orders, func() {
		renderOrderDetails(orders)
	})
	return nil
}

func buildOrderDetailsRequest(orderIDs []string) (service.OrderDetailsRequest, error) {
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return service.OrderDetailsRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	exchange := strings.ToUpper(strings.TrimSpace(orderFlags.exchange))
	if len(exchange) == 0 {
		return service.OrderDetailsRequest{}, fmt.Errorf("exchange is required")
	}
	return service.OrderDetailsRequest{
		Account:  account,
		Exchange: exchange,
		OrderIDs: orderIDs,
	}, nil
}

// runMaxOrderQty handles the max-qty subcommand.
func runMaxOrderQty(cmd *cobra.Command, args []string) error {
	req, err := buildTradingInfoRequest(args[0])
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	info, err := tradeSvc.GetTradingInfo(ctx, req)
	if err != nil {
		return fmt.Errorf("querying max order quantity: %w", err)
	}
	printResult(info, func() {
		renderTradingInfoTable(info)
	})
	return nil
}

func buildTradingInfoRequest(rawSymbol string) (service.TradingInfoRequest, error) {
	sym, err := symbol.Parse(rawSymbol)
	if err != nil {
		return service.TradingInfoRequest{}, fmt.Errorf("parsing symbol: %w", err)
	}
	account := resolveAccount(orderFlags.account)
	if len(account) == 0 {
		return service.TradingInfoRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	orderType, err := normalizeOrderType(orderFlags.typ)
	if err != nil {
		return service.TradingInfoRequest{}, err
	}
	price, err := parseOptionalPrice(orderFlags.price)
	if err != nil {
		return service.TradingInfoRequest{}, fmt.Errorf("invalid price: %w", err)
	}
	return service.TradingInfoRequest{
		Account:   account,
		Code:      sym.FutuCode(),
		OrderType: orderType,
		Price:     price,
		OrderID:   strings.TrimSpace(orderFlags.orderID),
	}, nil
}

func parseOptionalPrice(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return "", nil
	}
	price, err := strconv.ParseFloat(value, qtyParseBitSize)
	if err != nil || price <= 0 {
		return "", fmt.Errorf("must be a positive number")
	}
	return value, nil
}

func renderTradingInfoTable(info *service.TradingInfo) {
	tw := output.NewTableWriter(os.Stdout, tradingInfoHeaderKey, tradingInfoHeaderValue)
	tw.AddRow(tradingInfoLabelMaxCashBuy, string(info.MaxCashBuy))
	tw.AddRow(tradingInfoLabelMaxCashMarginBuy, string(info.MaxCashAndMarginBuy))
	tw.AddRow(tradingInfoLabelMaxPositionSell, string(info.MaxPositionSell))
	tw.AddRow(tradingInfoLabelMaxSellShort, string(info.MaxSellShort))
	tw.AddRow(tradingInfoLabelMaxBuyBack, string(info.MaxBuyBack))
	tw.AddRow(tradingInfoLabelLongRequiredIM, string(info.LongRequiredIM))
	tw.AddRow(tradingInfoLabelShortRequiredIM, string(info.ShortRequiredIM))
	tw.Render()
}

func renderOrderDetails(orders []service.Order) {
	for i := range orders {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		renderOrderDetail(&orders[i])
	}
}

func renderOrderDetail(order *service.Order) {
	tw := output.NewTableWriter(os.Stdout, orderDetailHeaderField, orderDetailHeaderValue)
	tw.AddRow("order_id", order.OrderID)
	tw.AddRow("symbol", order.Symbol)
	tw.AddRow("side", formatOrderValue(order.Side, orderSideLabels))
	tw.AddRow("type", formatOrderValue(order.Type, orderTypeLabels))
	tw.AddRow("price", fmt.Sprintf(orderPriceFormat, order.Price))
	tw.AddRow("qty", fmt.Sprintf("%d", order.Qty))
	tw.AddRow("filled_qty", fmt.Sprintf("%d", order.FilledQty))
	tw.AddRow("status", formatOrderValue(order.Status, orderStatusLabels))
	tw.AddRow("create_time", formatTimestamp(order.CreateTime))
	for _, key := range sortedOrderExtraKeys(order.Extra) {
		tw.AddRow(key, formatDetailValue(key, order.Extra[key]))
	}
	tw.Render()
}

func sortedOrderExtraKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatDetailValue(key string, value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if strings.HasSuffix(key, "_time") {
			return formatTimestamp(int64(typed))
		}
		return strconv.FormatFloat(typed, 'f', -1, qtyParseBitSize)
	case bool:
		return strconv.FormatBool(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

// renderOrderTable writes a list of orders as a formatted table to stdout.
func renderOrderTable(orders []service.Order) {
	tw := output.NewTableWriter(os.Stdout,
		orderHeaderID, orderHeaderSymbol, orderHeaderSide, orderHeaderType,
		orderHeaderPrice, orderHeaderQty, orderHeaderFilled, orderHeaderStatus,
		orderHeaderTime,
	)

	for i := range orders {
		tw.AddRow(orderToRow(&orders[i])...)
	}
	tw.Render()
}

// orderToRow converts an Order into a slice of formatted strings for table output.
func orderToRow(o *service.Order) []string {
	return []string{
		o.OrderID,
		o.Symbol,
		formatOrderValue(o.Side, orderSideLabels),
		formatOrderValue(o.Type, orderTypeLabels),
		fmt.Sprintf(orderPriceFormat, o.Price),
		fmt.Sprintf("%d", o.Qty),
		fmt.Sprintf("%d", o.FilledQty),
		formatOrderValue(o.Status, orderStatusLabels),
		formatTimestamp(o.CreateTime),
	}
}

func formatOrderValue(value string, labels map[string]string) string {
	if len(value) == 0 {
		return ""
	}
	label, ok := labels[value]
	if !ok {
		return value
	}
	return value + "(" + label + ")"
}
