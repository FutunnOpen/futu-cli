package cmd

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/client"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
)

const (
	simEndpointAccountList  = "sim_trade.account_list"
	simEndpointCashInfo     = "sim_trade.cash_info"
	simEndpointOrderList    = "sim_trade.order_list"
	simEndpointOrderHistory = "sim_trade.history_order_list"
	simEndpointMaxBuySell   = "sim_trade.max_buy_sell"
	simEndpointInputOrder   = "sim_trade.input_order"
	simEndpointModifyOrder  = "sim_trade.modify_order"
	simEndpointCancelOrder  = "sim_trade.cancel_order"
	simEndpointPositionList = "sim_trade.position_list"
)

const (
	simWrapAccounts  = "accounts"
	simWrapCashInfo  = "cash_info"
	simWrapOrders    = "orders"
	simWrapPositions = "positions"
)

const (
	simFieldAccountID         = "acc_id"
	simFieldMarket            = "market"
	simFieldSymbol            = "symbol"
	simFieldOrderType         = "order_type"
	simFieldOrderSide         = "order_side"
	simFieldOrderID           = "order_id"
	simFieldQuantity          = "qty"
	simFieldPrice             = "price"
	simFieldText              = "text"
	simFieldTradeTimeType     = "order_trade_time_type"
	simFieldTimeBegin         = "time_begin"
	simFieldTimeEnd           = "time_end"
	simFieldPageSize          = "page_size"
	simFieldNextKey           = "next_key"
	simFieldNewQuantity       = "new_qty"
	simFieldNewPrice          = "new_price"
	defaultSimMarket          = "HK"
	defaultSimOrderType       = "market"
	defaultSimHistoryPageSize = 50
	simMarketHongKong         = 1
	simMarketUnitedStates     = 2
	simMarketUSOption         = 3
	simMarketHongKongConnect  = 4
	simOrderTypeLimit         = 1
	simOrderTypeMarket        = 3
	simOrderSideBuyValue      = 1
	simOrderSideSellValue     = 2
	simTradeTimeTypeRegular   = 1
	simTradeTimeTypeExtended  = 2
	simQuantityParseBitSize   = 64
	simQuantityParseBase      = 10
	simInvalidOrderID         = "0"
	errSimModifyQuantityReq   = "--qty is required when modifying simulated orders"
	errSimModifyPriceReq      = "--price is required when modifying simulated orders"
)

var simFlags struct {
	account  string
	market   string
	price    string
	typ      string
	orderID  string
	symbol   string
	side     string
	quantity string
	text     string
	session  string
	pageSize int
	start    int64
	end      int64
	nextKey  string
}

var simCmd = &cobra.Command{
	Use:     "sim",
	Short:   "Simulated trading",
	Long:    "Manage simulated trading accounts, positions, and orders.",
	GroupID: simGroupID,
}

var simAccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Simulated account management",
	Long:  "List simulated accounts and query simulated account funds.",
}

var simAccountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List simulated accounts",
	Args:  cobra.NoArgs,
	RunE:  runSimAccountList,
}

var simAccountFundsCmd = &cobra.Command{
	Use:   "funds",
	Short: "Query simulated account funds",
	Args:  cobra.NoArgs,
	RunE:  runSimFunds,
}

var simPositionCmd = &cobra.Command{
	Use:   "position",
	Short: "List simulated positions",
	Args:  cobra.NoArgs,
	RunE:  runSimPositions,
}

var simOrderCmd = &cobra.Command{
	Use:   "order",
	Short: "Simulated order management",
	Long:  "Place, modify, cancel, and list simulated orders.",
}

var simOrderBuyCmd = &cobra.Command{
	Use:   "buy <symbol> <qty>",
	Short: "Place a simulated buy order",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSimPlaceOrder(cmd, args[0], args[1], orderSideBuy)
	},
}

var simOrderSellCmd = &cobra.Command{
	Use:   "sell <symbol> <qty>",
	Short: "Place a simulated sell order",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSimPlaceOrder(cmd, args[0], args[1], orderSideSell)
	},
}

var simOrderPlaceCmd = &cobra.Command{
	Use:   "place <qty>",
	Short: "Place a generic simulated order",
	Args:  cobra.ExactArgs(1),
	RunE:  runSimGenericPlaceOrder,
}

var simOrderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open simulated orders",
	Args:  cobra.NoArgs,
	RunE:  runSimOrderList,
}

var simOrderHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List historical simulated orders",
	Args:  cobra.NoArgs,
	RunE:  runSimOrderHistory,
}

var simOrderModifyCmd = &cobra.Command{
	Use:   "modify <order-id>",
	Short: "Modify a simulated order with required quantity and price",
	Args:  cobra.ExactArgs(1),
	RunE:  runSimModifyOrder,
}

var simOrderCancelCmd = &cobra.Command{
	Use:   "cancel <order-id>",
	Short: "Cancel a simulated order",
	Args:  cobra.ExactArgs(1),
	RunE:  runSimCancelOrder,
}

var simOrderMaxQtyCmd = &cobra.Command{
	Use:   "max-qty <symbol>",
	Short: "Query maximum simulated tradable quantity",
	Args:  cobra.ExactArgs(1),
	RunE:  runSimMaxQty,
}

func init() {
	registerSimAccountFlags(simAccountFundsCmd)
	registerSimMarketFlags(simPositionCmd)
	registerSimPlaceFlags(simOrderBuyCmd)
	registerSimPlaceFlags(simOrderSellCmd)
	registerSimGenericPlaceFlags(simOrderPlaceCmd)
	registerSimMarketFlags(simOrderListCmd)
	registerSimHistoryFlags(simOrderHistoryCmd)
	registerSimModifyFlags(simOrderModifyCmd)
	registerSimMarketFlags(simOrderCancelCmd)
	registerSimMaxQtyFlags(simOrderMaxQtyCmd)

	simAccountCmd.AddCommand(simAccountListCmd, simAccountFundsCmd)
	simOrderCmd.AddCommand(simOrderBuyCmd, simOrderSellCmd, simOrderPlaceCmd, simOrderModifyCmd, simOrderCancelCmd, simOrderListCmd, simOrderHistoryCmd, simOrderMaxQtyCmd)
	simCmd.AddCommand(simAccountCmd, simPositionCmd, simOrderCmd)
	rootCmd.AddCommand(simCmd)
}

func registerSimAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&simFlags.account, "account", "a", "", "Simulated account ID (default: first simulated account)")
}

func registerSimMarketFlags(cmd *cobra.Command) {
	registerSimAccountFlags(cmd)
	cmd.Flags().StringVarP(&simFlags.market, "market", "m", defaultSimMarket, "Simulated trading market: HK, US, US_OPTION, HKCC (default: HK)")
}

func registerSimPlaceFlags(cmd *cobra.Command) {
	registerSimMarketFlags(cmd)
	cmd.Flags().StringVar(&simFlags.price, "price", "", "Order price string (default: empty; omit for market orders)")
	cmd.Flags().StringVarP(&simFlags.typ, "type", "t", defaultSimOrderType, "Order type: market, limit (default: market)")
	cmd.Flags().StringVar(&simFlags.text, "text", "", "Order remark text (default: empty)")
	cmd.Flags().StringVar(&simFlags.session, "session", "", "Trading session: RTH, EXTENDED (default: backend default)")
}

func registerSimGenericPlaceFlags(cmd *cobra.Command) {
	registerSimPlaceFlags(cmd)
	cmd.Flags().StringVar(&simFlags.symbol, "symbol", "", "Symbol in MARKET.CODE format, for example US.AAPL")
	cmd.Flags().StringVar(&simFlags.side, "side", "", "Order side: BUY, SELL")
	_ = cmd.MarkFlagRequired("symbol")
	_ = cmd.MarkFlagRequired("side")
}

func registerSimHistoryFlags(cmd *cobra.Command) {
	registerSimMarketFlags(cmd)
	cmd.Flags().Int64Var(&simFlags.start, "start", 0, "Start time as a microsecond timestamp (default: no lower bound)")
	cmd.Flags().Int64Var(&simFlags.end, "end", 0, "End time as a microsecond timestamp (default: no upper bound)")
	cmd.Flags().IntVar(&simFlags.pageSize, "page-size", defaultSimHistoryPageSize, "Page size (default: 50)")
	cmd.Flags().StringVar(&simFlags.nextKey, "next-key", "", "Pagination cursor returned by the backend (default: empty)")
}

func registerSimModifyFlags(cmd *cobra.Command) {
	registerSimMarketFlags(cmd)
	cmd.Flags().StringVar(&simFlags.quantity, "qty", "", "New order quantity (required)")
	cmd.Flags().StringVar(&simFlags.price, "price", "", "New order price string (required)")
	_ = cmd.MarkFlagRequired("qty")
	_ = cmd.MarkFlagRequired("price")
}

func registerSimMaxQtyFlags(cmd *cobra.Command) {
	registerSimMarketFlags(cmd)
	cmd.Flags().StringVar(&simFlags.price, "price", "", "Reference price string (default: empty)")
	cmd.Flags().StringVarP(&simFlags.typ, "type", "t", defaultSimOrderType, "Order type: market, limit (default: market)")
	cmd.Flags().StringVar(&simFlags.orderID, "order-id", "", "Original order ID for modification scenarios (default: empty)")
}

func runSimAccountList(cmd *cobra.Command, _ []string) error {
	return callSimEndpoint(cmd, simEndpointAccountList, simWrapAccounts, client.CallOpts{})
}

func runSimFunds(cmd *cobra.Command, _ []string) error {
	opts, err := simPathOpts(cmd, "")
	if err != nil {
		return err
	}
	return callSimEndpoint(cmd, simEndpointCashInfo, simWrapCashInfo, opts)
}

func runSimPositions(cmd *cobra.Command, _ []string) error {
	opts, err := simQueryOpts(cmd)
	if err != nil {
		return err
	}
	return callSimEndpoint(cmd, simEndpointPositionList, simWrapPositions, opts)
}

func runSimOrderList(cmd *cobra.Command, _ []string) error {
	opts, err := simQueryOpts(cmd)
	if err != nil {
		return err
	}
	return callSimEndpoint(cmd, simEndpointOrderList, simWrapOrders, opts)
}

func runSimOrderHistory(cmd *cobra.Command, _ []string) error {
	opts, err := simQueryOpts(cmd)
	if err != nil {
		return err
	}
	applySimHistoryQuery(opts.Params)
	return callSimEndpoint(cmd, simEndpointOrderHistory, simWrapOrders, opts)
}

func runSimGenericPlaceOrder(cmd *cobra.Command, args []string) error {
	return runSimPlaceOrder(cmd, simFlags.symbol, args[0], simFlags.side)
}

func runSimPlaceOrder(cmd *cobra.Command, rawSymbol, rawQty, side string) error {
	opts, err := buildSimPlaceOrderOpts(cmd, rawSymbol, rawQty, side)
	if err != nil {
		return err
	}
	result, err := callSimEndpointResult(cmd, simEndpointInputOrder, opts)
	if err != nil {
		return err
	}
	if _, err := validateSimOrderResult(simEndpointInputOrder, result); err != nil {
		return err
	}
	printSimEndpointResult(result, "")
	return nil
}

func runSimModifyOrder(cmd *cobra.Command, args []string) error {
	opts, err := buildSimModifyOrderOpts(cmd, args[0])
	if err != nil {
		return err
	}
	result, err := callSimEndpointResult(cmd, simEndpointModifyOrder, opts)
	if err != nil {
		return err
	}
	orderID, err := validateSimOrderResult(simEndpointModifyOrder, result)
	if err != nil {
		return err
	}
	if err := verifySimOrderPresent(cmd, orderID); err != nil {
		return err
	}
	printSimEndpointResult(result, "")
	return nil
}

func runSimCancelOrder(cmd *cobra.Command, args []string) error {
	opts, err := buildSimCancelOrderOpts(cmd, args[0])
	if err != nil {
		return err
	}
	return callSimEndpoint(cmd, simEndpointCancelOrder, "", opts)
}

func runSimMaxQty(cmd *cobra.Command, args []string) error {
	opts, err := buildSimMaxQtyOpts(cmd, args[0])
	if err != nil {
		return err
	}
	return callSimEndpoint(cmd, simEndpointMaxBuySell, "", opts)
}

func buildSimPlaceOrderOpts(cmd *cobra.Command, rawSymbol, rawQty, side string) (client.CallOpts, error) {
	opts, err := simPathOpts(cmd, "")
	if err != nil {
		return client.CallOpts{}, err
	}
	body, err := buildSimPlaceOrderBody(rawSymbol, rawQty, side)
	if err != nil {
		return client.CallOpts{}, err
	}
	opts.Body = body
	return opts, nil
}

func buildSimPlaceOrderBody(rawSymbol, rawQty, side string) (map[string]any, error) {
	code, qty, err := parseSimOrderIdentity(rawSymbol, rawQty)
	if err != nil {
		return nil, err
	}
	body, err := simBaseOrderBody(side)
	if err != nil {
		return nil, err
	}
	body[simFieldSymbol], body[simFieldQuantity] = code, qty
	if strings.TrimSpace(simFlags.text) != "" {
		body[simFieldText] = strings.TrimSpace(simFlags.text)
	}
	return body, nil
}

func buildSimModifyOrderOpts(cmd *cobra.Command, orderID string) (client.CallOpts, error) {
	opts, err := simPathOpts(cmd, orderID)
	if err != nil {
		return client.CallOpts{}, err
	}
	body, err := simMarketBody()
	if err != nil {
		return client.CallOpts{}, err
	}
	if err := applySimModifyBody(body); err != nil {
		return client.CallOpts{}, err
	}
	opts.Body = body
	return opts, nil
}

func buildSimCancelOrderOpts(cmd *cobra.Command, orderID string) (client.CallOpts, error) {
	opts, err := simPathOpts(cmd, orderID)
	if err != nil {
		return client.CallOpts{}, err
	}
	opts.Body, err = simMarketBody()
	return opts, err
}

func buildSimMaxQtyOpts(cmd *cobra.Command, rawSymbol string) (client.CallOpts, error) {
	opts, err := simQueryOpts(cmd)
	if err != nil {
		return client.CallOpts{}, err
	}
	if err := applySimMaxQtyQuery(opts.Params, rawSymbol); err != nil {
		return client.CallOpts{}, err
	}
	return opts, nil
}

func simPathOpts(cmd *cobra.Command, orderID string) (client.CallOpts, error) {
	account, err := resolveSimAccount(cmd)
	if err != nil {
		return client.CallOpts{}, err
	}
	params := map[string]string{simFieldAccountID: account}
	if strings.TrimSpace(orderID) != "" {
		params[simFieldOrderID] = strings.TrimSpace(orderID)
	}
	return client.CallOpts{PathParams: params}, nil
}

func simQueryOpts(cmd *cobra.Command) (client.CallOpts, error) {
	opts, err := simPathOpts(cmd, "")
	if err != nil {
		return client.CallOpts{}, err
	}
	market, err := simMarketValue(simFlags.market)
	if err != nil {
		return client.CallOpts{}, err
	}
	opts.Params = url.Values{}
	opts.Params.Set(simFieldMarket, strconv.Itoa(market))
	return opts, nil
}

func simMarketBody() (map[string]any, error) {
	market, err := simMarketValue(simFlags.market)
	if err != nil {
		return nil, err
	}
	return map[string]any{simFieldMarket: market}, nil
}

func simBaseOrderBody(side string) (map[string]any, error) {
	body, err := simMarketBody()
	if err != nil {
		return nil, err
	}
	return applySimOrderFlags(body, side)
}

func applySimOrderFlags(body map[string]any, side string) (map[string]any, error) {
	orderSide, orderType, err := simOrderEnums(side)
	if err != nil {
		return nil, err
	}
	price, err := normalizePriceString(simFlags.price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	body[simFieldOrderSide], body[simFieldOrderType] = orderSide, orderType
	if price != "" {
		body[simFieldPrice] = price
	}
	return applySimSession(body)
}

func applySimSession(body map[string]any) (map[string]any, error) {
	session, ok, err := simSessionValue(simFlags.session)
	if err != nil {
		return nil, err
	}
	if ok {
		body[simFieldTradeTimeType] = session
	}
	return body, nil
}

func applySimModifyBody(body map[string]any) error {
	if strings.TrimSpace(simFlags.quantity) == "" {
		return errors.New(errSimModifyQuantityReq)
	}
	qty, err := parsePositiveSimQuantity(simFlags.quantity)
	if err != nil {
		return err
	}
	body[simFieldNewQuantity] = qty

	price, err := normalizePriceString(simFlags.price)
	if err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}
	if price == "" {
		return errors.New(errSimModifyPriceReq)
	}
	body[simFieldNewPrice] = price
	return nil
}

func applySimHistoryQuery(query url.Values) {
	if simFlags.start > 0 {
		query.Set(simFieldTimeBegin, strconv.FormatInt(simFlags.start, simQuantityParseBase))
	}
	if simFlags.end > 0 {
		query.Set(simFieldTimeEnd, strconv.FormatInt(simFlags.end, simQuantityParseBase))
	}
	if simFlags.pageSize > 0 {
		query.Set(simFieldPageSize, strconv.Itoa(simFlags.pageSize))
	}
	if strings.TrimSpace(simFlags.nextKey) != "" {
		query.Set(simFieldNextKey, strings.TrimSpace(simFlags.nextKey))
	}
}

func applySimMaxQtyQuery(query url.Values, rawSymbol string) error {
	code, err := parseSimSymbol(rawSymbol)
	if err != nil {
		return err
	}
	orderType, err := simOrderTypeValue(simFlags.typ)
	if err != nil {
		return err
	}
	query.Set(simFieldSymbol, code)
	query.Set(simFieldOrderType, strconv.Itoa(orderType))
	if strings.TrimSpace(simFlags.orderID) != "" {
		query.Set(simFieldOrderID, strings.TrimSpace(simFlags.orderID))
	}
	return applySimPriceQuery(query)
}

func applySimPriceQuery(query url.Values) error {
	price, err := normalizePriceString(simFlags.price)
	if err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}
	if price != "" {
		query.Set(simFieldPrice, price)
	}
	return nil
}

func resolveSimAccount(cmd *cobra.Command) (string, error) {
	account := strings.TrimSpace(simFlags.account)
	if account != "" {
		return account, nil
	}
	return firstSimAccountID(cmd)
}

func callSimEndpoint(cmd *cobra.Command, id, wrap string, opts client.CallOpts) error {
	result, err := callSimEndpointResult(cmd, id, opts)
	if err != nil {
		return err
	}
	printSimEndpointResult(result, wrap)
	return nil
}

func callSimEndpointResult(cmd *cobra.Command, id string, opts client.CallOpts) (*service.BFFResult, error) {
	result, err := quoteSvc.Call(cmdContext(cmd), id, opts)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", id, err)
	}
	return result, nil
}

func printSimEndpointResult(result *service.BFFResult, wrap string) {
	printBFFResult(result, func() {
		renderQuoteRESTTable(decodeJSONValue(result.Data), wrap)
	})
}

func validateSimOrderResult(endpoint string, result *service.BFFResult) (string, error) {
	orderID := strings.TrimSpace(jsonString(simOrderResultObject(result), simFieldOrderID))
	if orderID == "" || orderID == simInvalidOrderID {
		return "", fmt.Errorf("%s returned invalid %s %q; order may not have been submitted",
			endpoint, simFieldOrderID, orderID)
	}
	return orderID, nil
}

func simOrderResultObject(result *service.BFFResult) map[string]any {
	object, _ := decodeJSONValue(result.Data).(map[string]any)
	return object
}

func verifySimOrderPresent(cmd *cobra.Command, orderID string) error {
	opts, err := simQueryOpts(cmd)
	if err != nil {
		return err
	}
	result, err := callSimEndpointResult(cmd, simEndpointOrderList, opts)
	if err != nil {
		return err
	}
	if simOrderListContains(result, orderID) {
		return nil
	}
	return fmt.Errorf("simulated order %s was not found after modify; order may not have been modified", orderID)
}

func simOrderListContains(result *service.BFFResult, orderID string) bool {
	data := decodeJSONValue(result.Data)
	orders := jsonObjectList(data, simWrapOrders)
	if len(orders) == 0 {
		orders = jsonObjectList(data, "")
	}
	for _, order := range orders {
		if strings.TrimSpace(jsonString(order, simFieldOrderID)) == orderID {
			return true
		}
	}
	return false
}

func parseSimOrderIdentity(rawSymbol, rawQty string) (string, string, error) {
	code, err := parseSimSymbol(rawSymbol)
	if err != nil {
		return "", "", err
	}
	qty, err := parsePositiveSimQuantity(rawQty)
	if err != nil {
		return "", "", err
	}
	return code, qty, nil
}

func parseSimSymbol(rawSymbol string) (string, error) {
	sym, err := symbol.Parse(rawSymbol)
	if err != nil {
		return "", fmt.Errorf("parsing symbol: %w", err)
	}
	return sym.FutuCode(), nil
}

func parsePositiveSimQuantity(rawQty string) (string, error) {
	value := strings.TrimSpace(rawQty)
	qty, err := strconv.ParseFloat(value, simQuantityParseBitSize)
	if err != nil || qty <= 0 || math.IsInf(qty, 0) || math.IsNaN(qty) {
		return "", fmt.Errorf("quantity must be greater than 0")
	}
	return value, nil
}

func simOrderEnums(side string) (int, int, error) {
	orderSide, err := simOrderSideValue(side)
	if err != nil {
		return 0, 0, err
	}
	orderType, err := simOrderTypeValue(simFlags.typ)
	if err != nil {
		return 0, 0, err
	}
	return orderSide, orderType, nil
}

func simMarketValue(value string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "HK":
		return simMarketHongKong, nil
	case "US":
		return simMarketUnitedStates, nil
	case "US_OPTION", "OPTION":
		return simMarketUSOption, nil
	case "HKCC", "CN":
		return simMarketHongKongConnect, nil
	default:
		return 0, fmt.Errorf("unsupported simulated trading market %q", value)
	}
}

func simOrderTypeValue(value string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "MARKET":
		return simOrderTypeMarket, nil
	case "LIMIT":
		return simOrderTypeLimit, nil
	default:
		return 0, fmt.Errorf("unsupported simulated order type %q", value)
	}
}

func simOrderSideValue(value string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case orderSideBuy:
		return simOrderSideBuyValue, nil
	case orderSideSell:
		return simOrderSideSellValue, nil
	default:
		return 0, fmt.Errorf("unsupported simulated order side %q", value)
	}
}

func simSessionValue(value string) (int, bool, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "":
		return 0, false, nil
	case "RTH", "REGULAR":
		return simTradeTimeTypeRegular, true, nil
	case "EXTENDED", "PRE_POST", "ALL":
		return simTradeTimeTypeExtended, true, nil
	default:
		return 0, false, fmt.Errorf("unsupported simulated trading session %q", value)
	}
}
