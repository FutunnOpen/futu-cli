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

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
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
	"BUY":        "Buy",
	"SELL":       "Sell",
	"SELL_SHORT": "Sell Short",
	"BUY_BACK":   "Buy Back",
}

var orderTypeLabels = map[string]string{
	"LIMIT":             "Limit",
	"MARKET":            "Market",
	"AUCTION":           "Auction Market",
	"AUCTION_LIMIT":     "Auction Limit",
	"STOP":              "Stop",
	"STOP_LIMIT":        "Stop Limit",
	"MARKET_IF_TOUCHED": "Market If Touched",
	"LIMIT_IF_TOUCHED":  "Limit If Touched",
	orderTypeModified:   "Modified",
}

var orderStatusLabels = map[string]string{
	"WAITING_SUBMIT": "Waiting Submit",
	"SUBMITTING":     "Submitting",
	"SUBMITTED":      "Submitted",
	"FILLED_PART":    "Partially Filled",
	"FILLED_ALL":     "Filled",
	"CANCELLED_PART": "Partially Filled Then Cancelled",
	"CANCELLED_ALL":  "Cancelled",
	"FAILED":         "Failed",
	"DISABLED":       "Disabled",
	"DELETED":        "Deleted",
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
	Short: "Order management",
	Long:  "Manage trading orders: buy, sell, cancel, and list orders.",
}

var orderBuyCmd = &cobra.Command{
	Use:   "buy <symbol> <qty>",
	Short: "Place a buy order",
	Long:  "Place a buy order, supporting limit and market orders.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaceOrder(cmd, args, orderSideBuy)
	},
}

var orderSellCmd = &cobra.Command{
	Use:   "sell <symbol> <qty>",
	Short: "Place a sell order",
	Long:  "Place a sell order, supporting limit and market orders.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaceOrder(cmd, args, orderSideSell)
	},
}

var orderPlaceCmd = &cobra.Command{
	Use:   "place <qty>",
	Short: "Place a generic securities order",
	Long:  "Place a securities order supporting sell-short, buy-back, conditional, and multi-leg orders. Non-multi-leg orders require --symbol.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenericPlaceOrder,
}

var orderCancelCmd = &cobra.Command{
	Use:   "cancel <order-id>",
	Short: "Cancel an order",
	Long:  "Cancel the specified open order.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancelOrder,
}

var orderModifyCmd = &cobra.Command{
	Use:   "modify <order-id>",
	Short: "Modify an order",
	Long:  "Modify the quantity, price, or conditional trigger price of the specified order.",
	Args:  cobra.ExactArgs(1),
	RunE:  runModifyOrder,
}

var orderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open orders",
	Long:  "List open orders for the current account, including orders filled or cancelled in the last 24 hours.",
	Args:  cobra.NoArgs,
	RunE:  runListOrders,
}

var orderHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List historical orders",
	Long:  "List historical orders for the current account, with filters for market, time range, and symbol.",
	Args:  cobra.NoArgs,
	RunE:  runHistoryOrders,
}

var orderDetailCmd = &cobra.Command{
	Use:   "detail <order-id> [order-id...]",
	Short: "Show order details",
	Long:  "Get details for the specified order IDs. You can query up to 100 orders at once.",
	Args:  cobra.RangeArgs(1, maxOrderIDs),
	RunE:  runOrderDetails,
}

var orderMaxQtyCmd = &cobra.Command{
	Use:   "max-qty <symbol>",
	Short: "Query maximum tradable quantity",
	Long:  "Query maximum cash buy, margin buy, position sell, sell-short, and buy-back quantities for the specified symbol.",
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
	cmd.Flags().StringVar(&orderFlags.symbol, "symbol", "", "Symbol (default: empty; required for non-multi-leg orders)")
	cmd.Flags().StringVar(&orderFlags.side, "side", "", "Order side: BUY, SELL, SELL_SHORT, BUY_BACK (required; default: empty)")
	_ = cmd.MarkFlagRequired("side")
}

func registerPlaceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "Order price string (default: empty; omit for market orders)")
	cmd.Flags().StringVarP(&orderFlags.typ, "type", "t", defaultOrderType, "Order type (default: market)")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVar(&orderFlags.timeInForce, "time-in-force", defaultTimeInForce, "Time in force: DAY, GTC (default: DAY)")
	cmd.Flags().StringVar(&orderFlags.lotType, "lot-type", "", "HK stock lot type: ROUND, ODD (default: empty, decided by backend)")
	cmd.Flags().StringVar(&orderFlags.remark, "remark", "", "Order remark, up to 64 bytes (default: empty)")
	cmd.Flags().StringVar(&orderFlags.session, "session", "", "Trading session: RTH, RTH+Pre/Post-Mkt, OVERNIGHT, ALL_DAY (default: empty, decided by backend)")
	cmd.Flags().StringVar(&orderFlags.auxPrice, "aux-price", "", "Conditional trigger price string (default: empty)")
	cmd.Flags().StringVar(&orderFlags.orderClass, "order-class", "", "Order class: MLEG (default: empty for normal orders)")
	cmd.Flags().StringVar(&orderFlags.multiLegInfo, "multi-leg-info", "", "Multi-leg order JSON object (default: empty)")
}

// registerListOrderFlags adds flags for the list orders command.
func registerListOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVarP(&orderFlags.market, "market", "m", service.DefaultTradingMarket, "Trading market: HK, US, HKCC, SG, JP, AU, MY, KR (default: HK)")
	cmd.Flags().IntVar(&orderFlags.pageSize, "page-size", service.DefaultPageSize, "Page size, range 10-100 (default: 50)")
}

// registerHistoryOrderFlags adds flags for the history orders command.
func registerHistoryOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVarP(&orderFlags.market, "market", "m", service.DefaultTradingMarket, "Trading market: HK, US, HKCC, SG, JP, AU, MY, KR (default: HK)")
	cmd.Flags().Int64Var(&orderFlags.start, "start", 0, "Start time as a microsecond timestamp (default: no lower bound)")
	cmd.Flags().Int64Var(&orderFlags.end, "end", 0, "End time as a microsecond timestamp (default: no upper bound)")
	cmd.Flags().StringVar(&orderFlags.symbol, "symbol", "", "Symbol filter (default: no filter)")
	cmd.Flags().IntVar(&orderFlags.pageSize, "page-size", service.DefaultPageSize, "Page size, range 10-100 (default: 50)")
}

// registerDetailOrderFlags adds flags for the detail order command.
func registerDetailOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "Exchange code (required, for example US or SEHK)")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerModifyFlags adds flags required by the modify order command.
func registerModifyFlags(cmd *cobra.Command) {
	cmd.Flags().Int64Var(&orderFlags.qty, "qty", 0, "New order quantity (required)")
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "New order price string (required for limit orders; default: unchanged)")
	cmd.Flags().StringVar(&orderFlags.auxPrice, "aux-price", "", "New trigger price string for conditional orders (default: unchanged)")
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "Exchange code (required, for example HK or US)")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	_ = cmd.MarkFlagRequired("qty")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerCancelFlags adds flags required by the cancel order command.
func registerCancelFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.exchange, "exchange", "", "Exchange code (required, for example HK or US)")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	_ = cmd.MarkFlagRequired("exchange")
}

// registerMaxQtyFlags adds flags for the max-qty command.
func registerMaxQtyFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&orderFlags.price, "price", "", "Reference price string (required for non-market orders; default: empty)")
	cmd.Flags().StringVarP(&orderFlags.typ, "type", "t", defaultOrderType, "Order type: limit, market (default: market)")
	cmd.Flags().StringVar(&orderFlags.orderID, "order-id", "", "Original order ID for order modification scenarios (default: empty)")
	cmd.Flags().StringVarP(&orderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
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
