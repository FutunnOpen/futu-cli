package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/dryrun"
	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

const (
	cryptoBalanceHeaderSection = "Section"
	cryptoBalanceHeaderAsset   = "Asset"
	cryptoBalanceHeaderField   = "Field"
	cryptoBalanceHeaderValue   = "Value"
	cryptoBalanceSectionCash   = "Cash"
	cryptoBalanceSectionAsset  = "Asset"
	cryptoSideBuy              = "BUY"
	cryptoSideSell             = "SELL"
	cryptoMaxQtyLabelCashBuy   = "Max Cash Buy Qty"
	cryptoMaxQtyLabelSell      = "Max Sell Qty"
	cryptoMaxQtyLabelBuyAmount = "Max Cash Buy Amount"
	cryptoDealHeaderID         = "DealID"
	cryptoDealHeaderOrderID    = "OrderID"
	cryptoDealHeaderSymbol     = "Symbol"
	cryptoDealHeaderSide       = "Side"
	cryptoDealHeaderPrice      = "Price"
	cryptoDealHeaderQty        = "Qty"
	cryptoDealHeaderAmount     = "Amount"
	cryptoDealHeaderTime       = "Time"
	cryptoDealHeaderAvgPrice   = "AvgPrice"
	cryptoDealHeaderFirstTime  = "FirstTime"
	cryptoDealHeaderLastTime   = "LastTime"
	defaultCryptoOrderType     = "LIMIT"
	defaultCryptoTimeInForce   = "TIF_GTC"
	defaultCryptoCurrency      = "USD"
	cryptoReqIDRandomBytes     = 16
	cryptoTimestampParseBase   = 10
	cryptoTimestampParseBits   = 64
	defaultCryptoHistoryMonths = 3
	maxCryptoOrderIDs          = 20
)

var cryptoOrderFlags struct {
	account         string
	side            string
	orderType       string
	timeInForce     string
	price           string
	qty             string
	cashOrderQty    string
	currency        string
	coin            string
	expireTime      string
	conditionalInfo string
	reqID           string
	pageFlag        string
	pageSize        int
	start           int64
	end             int64
	symbol          string
	orderStatus     string
	orderVersion    int64
	groupByOrder    bool
	execute         string
}

type cryptoFillOrderSummary struct {
	OrderID   string
	Symbol    string
	Side      string
	Qty       float64
	Amount    float64
	FirstTime int64
	LastTime  int64
}

var cryptoCmd = &cobra.Command{
	Use:   "crypto",
	Short: "Cryptocurrency trading",
	Long:  "Manage cryptocurrency trading accounts, balances, orders, and fills.",
}

var cryptoAccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Cryptocurrency account management",
	Long:  "List authorized accounts that can be used for cryptocurrency trading.",
}

var cryptoAccountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cryptocurrency accounts",
	Long:  "List accounts available to the current user that have cryptocurrency trading enabled.",
	Args:  cobra.NoArgs,
	RunE:  runCryptoAccountList,
}

var cryptoBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Query cryptocurrency account balance",
	Long:  "Query total cryptocurrency account assets, including cash balances and digital asset positions.",
	Args:  cobra.NoArgs,
	RunE:  runCryptoBalance,
}

var cryptoMaxQtyCmd = &cobra.Command{
	Use:   "max-qty <symbol>",
	Short: "Query maximum cryptocurrency tradable quantity",
	Long:  "Query the maximum cash buy quantity, maximum sell quantity, and maximum cash buy amount for the specified cryptocurrency pair.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCryptoMaxQty,
}

var cryptoOrderCmd = &cobra.Command{
	Use:   "order",
	Short: "Cryptocurrency order management",
	Long:  "Place, modify, cancel, and query cryptocurrency orders.",
}

var cryptoOrderPlaceCmd = &cobra.Command{
	Use:   "place <symbol>",
	Short: "Place a cryptocurrency order",
	Long:  "Preview a cryptocurrency order by default. Re-run with --execute <confirmation_code> to submit. Supports limit, market, and conditional orders.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCryptoPlaceOrder(cmd, args[0], "")
	},
}

var cryptoOrderBuyCmd = &cobra.Command{
	Use:   "buy <symbol>",
	Short: "Place a cryptocurrency buy order",
	Long:  "Preview a cryptocurrency buy order by default. Re-run with --execute <confirmation_code> to submit.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCryptoPlaceOrder(cmd, args[0], cryptoSideBuy)
	},
}

var cryptoOrderSellCmd = &cobra.Command{
	Use:   "sell <symbol>",
	Short: "Place a cryptocurrency sell order",
	Long:  "Preview a cryptocurrency sell order by default. Re-run with --execute <confirmation_code> to submit.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCryptoPlaceOrder(cmd, args[0], cryptoSideSell)
	},
}

var cryptoOrderModifyCmd = &cobra.Command{
	Use:   "modify <order-id>",
	Short: "Modify a cryptocurrency order",
	Long:  "Preview a cryptocurrency order modification by default. Re-run with --execute <confirmation_code> to modify price, quantity, or conditional information.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCryptoModifyOrder,
}

var cryptoOrderCancelCmd = &cobra.Command{
	Use:   "cancel <order-id>",
	Short: "Cancel a cryptocurrency order",
	Long:  "Preview cancellation by default. Re-run with --execute <confirmation_code> to cancel the specified cryptocurrency order.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCryptoCancelOrder,
}

var cryptoOrderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open cryptocurrency orders",
	Long:  "List open cryptocurrency orders for the current account, with pagination support.",
	Args:  cobra.NoArgs,
	RunE:  runCryptoListOrders,
}

var cryptoOrderHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List historical cryptocurrency orders",
	Long:  "List historical cryptocurrency orders for the current account, with filters for time range, symbol, status, side, and order type.",
	Args:  cobra.NoArgs,
	RunE:  runCryptoHistoryOrders,
}

var cryptoOrderDetailCmd = &cobra.Command{
	Use:   "detail <order-id> [order-id...]",
	Short: "Show cryptocurrency order details",
	Long:  "Get details for the specified cryptocurrency order IDs. You can query up to 20 orders at once.",
	Args:  cobra.RangeArgs(1, maxCryptoOrderIDs),
	RunE:  runCryptoOrderDetails,
}

var cryptoDealCmd = &cobra.Command{
	Use:   "deal <order-id>",
	Short: "Query cryptocurrency fills",
	Long:  "Query fills for the specified cryptocurrency order, with pagination support.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCryptoFills,
}

var cryptoDealHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Query historical cryptocurrency fills",
	Long:  "Query historical cryptocurrency fills for the current account, with filters for time range, symbol, and pagination.",
	Args:  cobra.NoArgs,
	RunE:  runCryptoHistoryFills,
}

func init() {
	cryptoBalanceCmd.Flags().StringP("account", "a", "", "Trading account ID (default: config default_account)")
	registerCryptoMaxQtyFlags(cryptoMaxQtyCmd)
	registerCryptoOrderFlags(cryptoOrderPlaceCmd, true)
	registerCryptoOrderFlags(cryptoOrderBuyCmd, false)
	registerCryptoOrderFlags(cryptoOrderSellCmd, false)
	registerCryptoModifyOrderFlags(cryptoOrderModifyCmd)
	registerCryptoCancelOrderFlags(cryptoOrderCancelCmd)
	registerCryptoListOrderFlags(cryptoOrderListCmd)
	registerCryptoHistoryOrderFlags(cryptoOrderHistoryCmd)
	registerCryptoDetailOrderFlags(cryptoOrderDetailCmd)
	registerCryptoDealFlags(cryptoDealCmd)
	registerCryptoDealHistoryFlags(cryptoDealHistoryCmd)
	registerExecuteFlag(cryptoOrderPlaceCmd, &cryptoOrderFlags.execute)
	registerExecuteFlag(cryptoOrderBuyCmd, &cryptoOrderFlags.execute)
	registerExecuteFlag(cryptoOrderSellCmd, &cryptoOrderFlags.execute)
	registerExecuteFlag(cryptoOrderModifyCmd, &cryptoOrderFlags.execute)
	registerExecuteFlag(cryptoOrderCancelCmd, &cryptoOrderFlags.execute)
	cryptoAccountCmd.AddCommand(cryptoAccountListCmd)
	cryptoOrderCmd.AddCommand(
		cryptoOrderPlaceCmd,
		cryptoOrderBuyCmd,
		cryptoOrderSellCmd,
		cryptoOrderModifyCmd,
		cryptoOrderCancelCmd,
		cryptoOrderListCmd,
		cryptoOrderHistoryCmd,
		cryptoOrderDetailCmd,
	)
	cryptoDealCmd.AddCommand(cryptoDealHistoryCmd)
	cryptoCmd.AddCommand(cryptoAccountCmd, cryptoBalanceCmd, cryptoMaxQtyCmd, cryptoOrderCmd, cryptoDealCmd)
	rootCmd.AddCommand(cryptoCmd)
}

func registerCryptoMaxQtyFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVarP(&cryptoOrderFlags.orderType, "type", "t", defaultCryptoOrderType, "Order type (default: LIMIT)")
	cmd.Flags().StringVar(&cryptoOrderFlags.currency, "currency", defaultCryptoCurrency, "Settlement currency (default: USD)")
	cmd.Flags().StringVar(&cryptoOrderFlags.coin, "coin", "", "Base coin (default: inferred from symbol)")
	cmd.Flags().StringVar(&cryptoOrderFlags.price, "price", "", "Reference price (default: empty; required and must be positive for limit-like orders)")
}

func registerCryptoOrderFlags(cmd *cobra.Command, includeSide bool) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	if includeSide {
		cmd.Flags().StringVar(&cryptoOrderFlags.side, "side", "", "Order side: BUY, SELL (required)")
		_ = cmd.MarkFlagRequired("side")
	}
	cmd.Flags().StringVar(&cryptoOrderFlags.reqID, "req-id", "", "Idempotency ID (default: auto-generated)")
	cmd.Flags().StringVarP(&cryptoOrderFlags.orderType, "type", "t", defaultCryptoOrderType, "Order type (default: LIMIT)")
	cmd.Flags().StringVar(&cryptoOrderFlags.timeInForce, "time-in-force", defaultCryptoTimeInForce, "Time in force (default: TIF_GTC)")
	cmd.Flags().StringVar(&cryptoOrderFlags.price, "price", "", "Order price (default: empty; required for limit-like orders)")
	cmd.Flags().StringVar(&cryptoOrderFlags.qty, "qty", "", "Coin quantity (default: empty; mutually exclusive with --cash-qty)")
	cmd.Flags().StringVar(&cryptoOrderFlags.cashOrderQty, "cash-qty", "", "Cash amount order quantity (default: empty; mutually exclusive with --qty; not supported for sell orders)")
	cmd.Flags().StringVar(&cryptoOrderFlags.expireTime, "expire-time", "", "GTD order expiration time (default: empty)")
	cmd.Flags().StringVar(&cryptoOrderFlags.conditionalInfo, "conditional-info", "", "Conditional order JSON (default: empty)")
}

func registerCryptoModifyOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVar(&cryptoOrderFlags.reqID, "req-id", "", "Idempotency ID (default: auto-generated)")
	cmd.Flags().StringVar(&cryptoOrderFlags.price, "price", "", "Updated order price (required; use 0 for stop-profit/stop-loss market orders)")
	cmd.Flags().StringVar(&cryptoOrderFlags.qty, "qty", "", "Updated order quantity (required)")
	cmd.Flags().Int64Var(&cryptoOrderFlags.orderVersion, "order-version", 0, "Order version (required; from order detail version)")
	cmd.Flags().StringVar(&cryptoOrderFlags.conditionalInfo, "conditional-info", "", "Conditional order JSON (required)")
	_ = cmd.MarkFlagRequired("price")
	_ = cmd.MarkFlagRequired("qty")
	_ = cmd.MarkFlagRequired("order-version")
	_ = cmd.MarkFlagRequired("conditional-info")
}

func registerCryptoCancelOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringVar(&cryptoOrderFlags.reqID, "req-id", "", "Idempotency ID (default: auto-generated)")
}

func registerCryptoListOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().IntVar(&cryptoOrderFlags.pageSize, "page-size", service.DefaultCryptoPageSize, "Page size, max 50 (default: 50)")
	cmd.Flags().StringVar(&cryptoOrderFlags.pageFlag, "page-flag", "", "Pagination cursor (default: empty, first page)")
}

func registerCryptoHistoryOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().Int64Var(&cryptoOrderFlags.start, "start", 0, "Start time as a microsecond timestamp (default: last 3 months)")
	cmd.Flags().Int64Var(&cryptoOrderFlags.end, "end", 0, "End time as a microsecond timestamp (default: now)")
	cmd.Flags().IntVar(&cryptoOrderFlags.pageSize, "page-size", service.DefaultCryptoPageSize, "Page size, max 50 (default: 50)")
	cmd.Flags().StringVar(&cryptoOrderFlags.pageFlag, "page-flag", "", "Pagination cursor (default: empty, first page)")
	cmd.Flags().StringVar(&cryptoOrderFlags.symbol, "symbol", "", "Symbol filter, for example BTCUSD (default: no filter)")
	cmd.Flags().StringVar(&cryptoOrderFlags.orderStatus, "order-status", "", "Order status filter; comma-separated values supported (default: no filter)")
	cmd.Flags().StringVar(&cryptoOrderFlags.currency, "currency", "", "Settlement currency filter; comma-separated values supported (default: no filter)")
	cmd.Flags().StringVar(&cryptoOrderFlags.side, "side", "", "Order side filter; comma-separated values supported: BUY, SELL (default: no filter)")
	cmd.Flags().StringVar(&cryptoOrderFlags.orderType, "type", "", "Order type filter; comma-separated values supported (default: no filter)")
}

func registerCryptoDetailOrderFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
}

func registerCryptoDealFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().IntVar(&cryptoOrderFlags.pageSize, "page-size", service.DefaultCryptoPageSize, "Page size, max 50 (default: 50)")
	cmd.Flags().StringVar(&cryptoOrderFlags.pageFlag, "page-flag", "", "Pagination cursor (default: empty, first page)")
	cmd.Flags().BoolVar(&cryptoOrderFlags.groupByOrder, "group-by-order", false, "Group fills by order (default: false)")
}

func registerCryptoDealHistoryFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cryptoOrderFlags.account, "account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().Int64Var(&cryptoOrderFlags.start, "start", 0, "Start time as a microsecond timestamp (default: last 3 months)")
	cmd.Flags().Int64Var(&cryptoOrderFlags.end, "end", 0, "End time as a microsecond timestamp (default: now)")
	cmd.Flags().IntVar(&cryptoOrderFlags.pageSize, "page-size", service.DefaultCryptoPageSize, "Page size, max 50 (default: 50)")
	cmd.Flags().StringVar(&cryptoOrderFlags.pageFlag, "page-flag", "", "Pagination cursor (default: empty, first page)")
	cmd.Flags().StringVar(&cryptoOrderFlags.symbol, "symbol", "", "Symbol filter, for example BTCUSD (default: no filter)")
	cmd.Flags().BoolVar(&cryptoOrderFlags.groupByOrder, "group-by-order", false, "Group fills by order (default: false)")
}

func runCryptoAccountList(cmd *cobra.Command, _ []string) error {
	ctx := cmdContext(cmd)
	accounts, err := tradeSvc.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("fetching crypto accounts: %w", err)
	}
	cryptoAccounts := filterAccountsByMarket(accounts, marketCrypto)
	printResult(cryptoAccounts, func() {
		renderAccountTable(cryptoAccounts)
	})
	return nil
}

func runCryptoBalance(cmd *cobra.Command, _ []string) error {
	account, err := cmd.Flags().GetString("account")
	if err != nil {
		return fmt.Errorf("reading account flag: %w", err)
	}
	account = resolveAccount(account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	ctx := cmdContext(cmd)
	balance, err := cryptoSvc.GetTotalBalance(ctx, account)
	if err != nil {
		return fmt.Errorf("fetching crypto balance: %w", err)
	}
	printResult(balance, func() {
		renderCryptoBalanceTable(balance)
	})
	return nil
}

func runCryptoMaxQty(cmd *cobra.Command, args []string) error {
	req, err := buildCryptoMaxQtyRequest(args[0])
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	maxQty, err := cryptoSvc.GetMaxQty(ctx, req)
	if err != nil {
		return fmt.Errorf("querying crypto max quantity: %w", err)
	}
	printResult(maxQty, func() {
		renderCryptoMaxQtyTable(maxQty)
	})
	return nil
}

func runCryptoPlaceOrder(cmd *cobra.Command, symbol string, fixedSide string) error {
	req, err := buildCryptoPlaceOrderRequest(symbol, fixedSide)
	if err != nil {
		return err
	}
	proceed, err := gateTradeExecution(
		dryrun.Order(req.Side, req.Symbol, cryptoOrderQuantity(req), req.Price),
		cryptoOrderFlags.execute,
		cryptoPlaceOrderPreview(req),
	)
	if err != nil || !proceed {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.PlaceOrder(ctx, req)
	if err != nil {
		return fmt.Errorf("placing crypto order: %w", err)
	}
	printResult(result, func() {
		renderCryptoOrderResult(result)
	})
	return nil
}

func runCryptoModifyOrder(cmd *cobra.Command, args []string) error {
	req, err := buildCryptoModifyOrderRequest()
	if err != nil {
		return err
	}
	proceed, err := gateTradeExecution(
		dryrun.Replace(args[0], req.Qty, req.Price),
		cryptoOrderFlags.execute,
		map[string]any{
			"action":        "modify",
			"order_id":      args[0],
			"qty":           req.Qty,
			"price":         req.Price,
			"order_version": req.OrderVersion,
			"account":       req.Account,
		},
	)
	if err != nil || !proceed {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.ModifyOrder(ctx, args[0], req)
	if err != nil {
		return fmt.Errorf("modifying crypto order: %w", err)
	}
	printResult(result, func() {
		renderCryptoModifyResult(args[0], result)
	})
	return nil
}

func runCryptoCancelOrder(cmd *cobra.Command, args []string) error {
	req, err := buildCryptoCancelOrderRequest()
	if err != nil {
		return err
	}
	proceed, err := gateTradeExecution(
		dryrun.OnOrder("cancel", args[0]),
		cryptoOrderFlags.execute,
		map[string]any{
			"action":   "cancel",
			"order_id": args[0],
			"account":  req.Account,
		},
	)
	if err != nil || !proceed {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.CancelOrder(ctx, args[0], req)
	if err != nil {
		return fmt.Errorf("cancelling crypto order: %w", err)
	}
	printResult(result, func() {
		renderCryptoCancelResult(args[0], result)
	})
	return nil
}

func cryptoOrderQuantity(req service.CryptoPlaceOrderRequest) string {
	if len(strings.TrimSpace(req.Qty)) > 0 {
		return req.Qty
	}
	return req.CashOrderQty
}

func cryptoPlaceOrderPreview(req service.CryptoPlaceOrderRequest) map[string]any {
	return map[string]any{
		"side":          req.Side,
		"symbol":        req.Symbol,
		"qty":           req.Qty,
		"cash_qty":      req.CashOrderQty,
		"type":          req.OrderType,
		"price":         req.Price,
		"account":       req.Account,
		"time_in_force": req.TimeInForce,
	}
}

func runCryptoListOrders(cmd *cobra.Command, _ []string) error {
	req, err := buildCryptoOrdersRequest()
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.ListActiveOrders(ctx, req)
	if err != nil {
		return fmt.Errorf("listing crypto active orders: %w", err)
	}
	printResult(result, func() {
		renderCryptoOrderListResult(result)
	})
	return nil
}

func runCryptoHistoryOrders(cmd *cobra.Command, _ []string) error {
	req, err := buildCryptoHistoryOrdersRequest()
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.ListHistoryOrders(ctx, req)
	if err != nil {
		return fmt.Errorf("listing crypto history orders: %w", err)
	}
	printResult(result, func() {
		renderCryptoOrderListResult(result)
	})
	return nil
}

func runCryptoOrderDetails(cmd *cobra.Command, args []string) error {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	ctx := cmdContext(cmd)
	orders, err := cryptoSvc.GetOrderDetails(ctx, account, args)
	if err != nil {
		return fmt.Errorf("getting crypto order details: %w", err)
	}
	printResult(orders, func() {
		renderCryptoOrderDetails(orders)
	})
	return nil
}

func runCryptoFills(cmd *cobra.Command, args []string) error {
	req, err := buildCryptoFillsRequest(args[0])
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.ListFills(ctx, req)
	if err != nil {
		return fmt.Errorf("listing crypto fills: %w", err)
	}
	printResult(result, func() {
		renderCryptoFillListResult(result)
	})
	return nil
}

func runCryptoHistoryFills(cmd *cobra.Command, _ []string) error {
	req, err := buildCryptoHistoryFillsRequest()
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	result, err := cryptoSvc.ListHistoryFills(ctx, req)
	if err != nil {
		return fmt.Errorf("listing crypto history fills: %w", err)
	}
	printResult(result, func() {
		renderCryptoFillListResult(result)
	})
	return nil
}

func buildCryptoMaxQtyRequest(symbol string) (service.CryptoMaxQtyRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoMaxQtyRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	normalizedSymbol := normalizeCryptoSymbol(symbol)
	return service.CryptoMaxQtyRequest{
		Account:   account,
		OrderType: strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.orderType)),
		Currency:  strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.currency)),
		Coin:      cryptoMaxQtyCoin(normalizedSymbol),
		Price:     strings.TrimSpace(cryptoOrderFlags.price),
	}, nil
}

func cryptoMaxQtyCoin(symbol string) string {
	coin := strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.coin))
	if len(coin) > 0 {
		return coin
	}
	return inferCryptoCoin(symbol, strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.currency)))
}

func inferCryptoCoin(symbol string, currency string) string {
	symbol = normalizeCryptoSymbol(symbol)
	if len(currency) > 0 && strings.HasSuffix(symbol, currency) {
		return strings.TrimSuffix(symbol, currency)
	}
	return symbol
}

func buildCryptoOrdersRequest() (service.CryptoOrdersRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoOrdersRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	return service.CryptoOrdersRequest{
		Account:  account,
		PageSize: cryptoOrderFlags.pageSize,
		PageFlag: strings.TrimSpace(cryptoOrderFlags.pageFlag),
	}, nil
}

func buildCryptoHistoryOrdersRequest() (service.CryptoOrdersRequest, error) {
	req, err := buildCryptoOrdersRequest()
	if err != nil {
		return service.CryptoOrdersRequest{}, err
	}
	req.Start, req.End = cryptoHistoryTimeRange()
	req.Symbol = normalizeCryptoSymbol(cryptoOrderFlags.symbol)
	req.OrderStatus = strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.orderStatus))
	req.Currency = strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.currency))
	req.Side = strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.side))
	req.OrderType = strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.orderType))
	return req, nil
}

func buildCryptoFillsRequest(orderID string) (service.CryptoFillsRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoFillsRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	return service.CryptoFillsRequest{
		Account:  account,
		OrderID:  strings.TrimSpace(orderID),
		PageSize: cryptoOrderFlags.pageSize,
		PageFlag: strings.TrimSpace(cryptoOrderFlags.pageFlag),
	}, nil
}

func buildCryptoHistoryFillsRequest() (service.CryptoFillsRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoFillsRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	start, end := cryptoHistoryTimeRange()
	return service.CryptoFillsRequest{
		Account:  account,
		Start:    start,
		End:      end,
		Symbol:   normalizeCryptoSymbol(cryptoOrderFlags.symbol),
		PageSize: cryptoOrderFlags.pageSize,
		PageFlag: strings.TrimSpace(cryptoOrderFlags.pageFlag),
	}, nil
}

func cryptoHistoryTimeRange() (int64, int64) {
	end := cryptoOrderFlags.end
	if end == 0 {
		end = time.Now().UnixMicro()
	}
	start := cryptoOrderFlags.start
	if start == 0 {
		start = time.UnixMicro(end).AddDate(0, -defaultCryptoHistoryMonths, 0).UnixMicro()
	}
	return start, end
}

func buildCryptoPlaceOrderRequest(symbol string, fixedSide string) (service.CryptoPlaceOrderRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoPlaceOrderRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	reqID, err := cryptoOrderReqID()
	if err != nil {
		return service.CryptoPlaceOrderRequest{}, err
	}
	conditionalInfo, err := service.DecodeConditionalInfo(strings.TrimSpace(cryptoOrderFlags.conditionalInfo))
	if err != nil {
		return service.CryptoPlaceOrderRequest{}, err
	}
	return newCryptoPlaceOrderRequest(account, symbol, fixedSide, reqID, conditionalInfo), nil
}

func newCryptoPlaceOrderRequest(
	account string, symbol string, fixedSide string, reqID string, conditionalInfo map[string]any,
) service.CryptoPlaceOrderRequest {
	side := strings.ToUpper(strings.TrimSpace(fixedSide))
	if len(side) == 0 {
		side = strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.side))
	}
	return service.CryptoPlaceOrderRequest{
		Account:         account,
		ReqID:           reqID,
		Side:            side,
		Symbol:          normalizeCryptoSymbol(symbol),
		OrderType:       strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.orderType)),
		TimeInForce:     strings.ToUpper(strings.TrimSpace(cryptoOrderFlags.timeInForce)),
		Price:           strings.TrimSpace(cryptoOrderFlags.price),
		Qty:             strings.TrimSpace(cryptoOrderFlags.qty),
		CashOrderQty:    strings.TrimSpace(cryptoOrderFlags.cashOrderQty),
		ExpireTime:      strings.TrimSpace(cryptoOrderFlags.expireTime),
		ConditionalInfo: conditionalInfo,
	}
}

func buildCryptoModifyOrderRequest() (service.CryptoModifyOrderRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoModifyOrderRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	reqID, err := cryptoOrderReqID()
	if err != nil {
		return service.CryptoModifyOrderRequest{}, err
	}
	conditionalInfo, err := service.DecodeConditionalInfo(strings.TrimSpace(cryptoOrderFlags.conditionalInfo))
	if err != nil {
		return service.CryptoModifyOrderRequest{}, err
	}
	return service.CryptoModifyOrderRequest{
		Account:         account,
		ReqID:           reqID,
		Price:           strings.TrimSpace(cryptoOrderFlags.price),
		Qty:             strings.TrimSpace(cryptoOrderFlags.qty),
		OrderVersion:    cryptoOrderFlags.orderVersion,
		ConditionalInfo: conditionalInfo,
	}, nil
}

func buildCryptoCancelOrderRequest() (service.CryptoCancelOrderRequest, error) {
	account := resolveAccount(cryptoOrderFlags.account)
	if len(account) == 0 {
		return service.CryptoCancelOrderRequest{}, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	reqID, err := cryptoOrderReqID()
	if err != nil {
		return service.CryptoCancelOrderRequest{}, err
	}
	return service.CryptoCancelOrderRequest{
		Account: account,
		ReqID:   reqID,
	}, nil
}

func normalizeCryptoSymbol(symbol string) string {
	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(symbol)), "CC.")
}

func cryptoOrderReqID() (string, error) {
	reqID := strings.TrimSpace(cryptoOrderFlags.reqID)
	if len(reqID) > 0 {
		return reqID, nil
	}
	var data [cryptoReqIDRandomBytes]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate req-id: %w", err)
	}
	return fmt.Sprintf("%d_%s", cmdStartTime.UnixMicro(), hex.EncodeToString(data[:])), nil
}

func renderCryptoOrderResult(result *service.CryptoOrderResult) {
	tw := output.NewTableWriter(os.Stdout, orderDetailHeaderField, orderDetailHeaderValue)
	tw.AddRow("order_id", result.OrderID)
	tw.Render()
}

func renderCryptoModifyResult(orderID string, result *service.CryptoOrderResult) {
	fmt.Fprintf(os.Stdout, "Order modified: %s\n", cryptoMutationOrderID(orderID, result))
}

func renderCryptoCancelResult(orderID string, result *service.CryptoOrderResult) {
	fmt.Fprintf(os.Stdout, "Order cancelled: %s\n", cryptoMutationOrderID(orderID, result))
}

func cryptoMutationOrderID(orderID string, result *service.CryptoOrderResult) string {
	if result != nil && len(result.OrderID) > 0 {
		return result.OrderID
	}
	return orderID
}

func renderCryptoMaxQtyTable(maxQty *service.CryptoMaxQty) {
	tw := output.NewTableWriter(os.Stdout, orderDetailHeaderField, orderDetailHeaderValue)
	tw.AddRow(cryptoMaxQtyLabelCashBuy, maxQty.MaxCashBuyQty)
	tw.AddRow(cryptoMaxQtyLabelSell, maxQty.MaxSellQty)
	tw.AddRow(cryptoMaxQtyLabelBuyAmount, maxQty.MaxCashBuyAmount)
	tw.Render()
}

func renderCryptoOrderTable(orders []service.CryptoOrder) {
	tw := output.NewTableWriter(os.Stdout,
		orderHeaderID,
		orderHeaderSymbol,
		orderHeaderSide,
		orderHeaderType,
		orderHeaderPrice,
		orderHeaderQty,
		orderHeaderFilled,
		orderHeaderStatus,
		orderHeaderTime,
	)
	for _, order := range orders {
		tw.AddRow(
			cryptoOrderField(order, "order_id"),
			cryptoOrderSymbol(order),
			formatOrderValue(cryptoOrderField(order, "side", "trd_side"), orderSideLabels),
			formatOrderValue(cryptoOrderField(order, "ord_type", "order_type", "type"), orderTypeLabels),
			cryptoOrderField(order, "price"),
			cryptoOrderField(order, "qty", "quantity", "order_qty", "order_quantity"),
			cryptoOrderField(order, "filled_qty", "dealt_qty", "filled_quantity", "dealt_quantity", "cum_qty"),
			formatOrderValue(cryptoOrderField(order, "status", "order_status", "ord_status", "order_state"), orderStatusLabels),
			formatCryptoOrderTime(order),
		)
	}
	tw.Render()
}

func renderCryptoOrderListResult(result *service.CryptoOrderList) {
	if len(result.Orders) == 0 {
		fmt.Fprintln(os.Stdout, "No orders found.")
		return
	}
	renderCryptoOrderTable(result.Orders)
	renderCryptoPageInfo(result)
}

func renderCryptoFillListResult(result *service.CryptoFillList) {
	if len(result.OrderFills) == 0 {
		fmt.Fprintln(os.Stdout, "No fills found.")
		return
	}
	if cryptoOrderFlags.groupByOrder {
		renderCryptoFillSummaryTable(result.OrderFills)
		renderCryptoFillPageInfo(result)
		return
	}
	renderCryptoFillTable(result.OrderFills)
	renderCryptoFillPageInfo(result)
}

func renderCryptoFillTable(fills []service.CryptoFill) {
	tw := output.NewTableWriter(os.Stdout,
		cryptoDealHeaderID,
		cryptoDealHeaderOrderID,
		cryptoDealHeaderSymbol,
		cryptoDealHeaderSide,
		cryptoDealHeaderPrice,
		cryptoDealHeaderQty,
		cryptoDealHeaderAmount,
		cryptoDealHeaderTime,
	)
	for _, fill := range fills {
		tw.AddRow(
			cryptoFillField(fill, "fill_id", "deal_id", "id"),
			cryptoFillField(fill, "order_id", "ord_id", "order_show_id"),
			cryptoFillSymbol(fill),
			formatOrderValue(cryptoFillField(fill, "side", "trd_side"), orderSideLabels),
			cryptoFillField(fill, "price", "fill_price", "deal_price"),
			cryptoFillField(fill, "qty", "quantity", "fill_qty", "deal_qty"),
			cryptoFillField(fill, "amount", "fill_amount", "deal_amount", "exec_amount"),
			formatCryptoFillTime(fill),
		)
	}
	tw.Render()
}

func renderCryptoFillSummaryTable(fills []service.CryptoFill) {
	summaries := summarizeCryptoFillsByOrder(fills)
	tw := output.NewTableWriter(os.Stdout,
		cryptoDealHeaderOrderID,
		cryptoDealHeaderSymbol,
		cryptoDealHeaderSide,
		cryptoDealHeaderQty,
		cryptoDealHeaderAmount,
		cryptoDealHeaderAvgPrice,
		cryptoDealHeaderFirstTime,
		cryptoDealHeaderLastTime,
	)
	for _, summary := range summaries {
		tw.AddRow(
			summary.OrderID,
			summary.Symbol,
			formatOrderValue(summary.Side, orderSideLabels),
			formatCryptoFloat(summary.Qty),
			formatCryptoFloat(summary.Amount),
			formatCryptoAveragePrice(summary),
			formatCryptoSummaryTime(summary.FirstTime),
			formatCryptoSummaryTime(summary.LastTime),
		)
	}
	tw.Render()
}

func summarizeCryptoFillsByOrder(fills []service.CryptoFill) []cryptoFillOrderSummary {
	summaryByOrder := make(map[string]*cryptoFillOrderSummary)
	orderIDs := make([]string, 0, len(fills))
	for _, fill := range fills {
		orderID := cryptoFillField(fill, "order_id", "ord_id", "order_show_id")
		summary := cryptoFillSummary(summaryByOrder, orderID, fill)
		if _, ok := summaryByOrder[orderID]; !ok {
			summaryByOrder[orderID] = summary
			orderIDs = append(orderIDs, orderID)
		}
		mergeCryptoFillSummary(summary, fill)
	}
	return cryptoFillSummaries(summaryByOrder, orderIDs)
}

func cryptoFillSummary(
	summaryByOrder map[string]*cryptoFillOrderSummary,
	orderID string,
	fill service.CryptoFill,
) *cryptoFillOrderSummary {
	if summary, ok := summaryByOrder[orderID]; ok {
		return summary
	}
	return &cryptoFillOrderSummary{
		OrderID: orderID,
		Symbol:  cryptoFillSymbol(fill),
		Side:    cryptoFillField(fill, "side", "trd_side"),
	}
}

func mergeCryptoFillSummary(summary *cryptoFillOrderSummary, fill service.CryptoFill) {
	summary.Qty += parseCryptoFloat(cryptoFillField(fill, "qty", "quantity", "fill_qty", "deal_qty"))
	summary.Amount += parseCryptoFloat(cryptoFillField(fill, "amount", "fill_amount", "deal_amount", "exec_amount"))
	fillTime := cryptoFillTimestamp(fill)
	if fillTime <= 0 {
		return
	}
	if summary.FirstTime == 0 || fillTime < summary.FirstTime {
		summary.FirstTime = fillTime
	}
	if fillTime > summary.LastTime {
		summary.LastTime = fillTime
	}
}

func cryptoFillSummaries(
	summaryByOrder map[string]*cryptoFillOrderSummary,
	orderIDs []string,
) []cryptoFillOrderSummary {
	summaries := make([]cryptoFillOrderSummary, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		summaries = append(summaries, *summaryByOrder[orderID])
	}
	return summaries
}

func renderCryptoFillPageInfo(result *service.CryptoFillList) {
	if result.Completed || len(result.PageFlag) == 0 || result.PageFlag == "0" {
		return
	}
	fmt.Fprintf(os.Stdout, "NextPageFlag: %s\nCompleted: %t\n", result.PageFlag, result.Completed)
}

func renderCryptoOrderDetails(orders []service.CryptoOrder) {
	if len(orders) == 0 {
		fmt.Fprintln(os.Stdout, "No orders found.")
		return
	}
	for i := range orders {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		renderCryptoOrderDetail(orders[i])
	}
}

func renderCryptoOrderDetail(order service.CryptoOrder) {
	tw := output.NewTableWriter(os.Stdout, orderDetailHeaderField, orderDetailHeaderValue)
	tw.AddRow("order_id", cryptoOrderField(order, "order_id"))
	tw.AddRow("symbol", cryptoOrderSymbol(order))
	for _, key := range sortedCryptoOrderDetailKeys(order) {
		value := formatCryptoDetailValue(key, order[key])
		if len(value) == 0 {
			continue
		}
		tw.AddRow(key, value)
	}
	tw.Render()
}

func sortedCryptoOrderDetailKeys(order service.CryptoOrder) []string {
	keys := make([]string, 0, len(order))
	for key := range order {
		if key == "order_id" || key == "symbol" || key == "symbol_pair" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderCryptoPageInfo(result *service.CryptoOrderList) {
	if result.Completed || len(result.PageFlag) == 0 || result.PageFlag == "0" {
		return
	}
	fmt.Fprintf(os.Stdout, "NextPageFlag: %s\nCompleted: %t\n", result.PageFlag, result.Completed)
}

func cryptoOrderField(order service.CryptoOrder, keys ...string) string {
	for _, key := range keys {
		value := formatCryptoOrderFieldValue(order[key])
		if len(value) > 0 && value != "<nil>" {
			return value
		}
	}
	return "-"
}

func cryptoOrderSymbol(order service.CryptoOrder) string {
	symbol := cryptoOrderField(order, "symbol", "code", "trade_pair", "trading_pair")
	if symbol != "-" {
		return symbol
	}
	symbol = cryptoOrderNestedField(order, "symbol_pair", "symbol")
	if symbol != "-" {
		return symbol
	}
	coin := cryptoOrderField(order, "coin", "base_currency", "base_coin")
	currency := cryptoOrderField(order, "currency", "quote_currency", "quote_coin")
	if coin == "-" {
		coin = cryptoOrderNestedField(order, "symbol_pair", "base")
	}
	if currency == "-" {
		currency = cryptoOrderNestedField(order, "symbol_pair", "quote")
	}
	if coin != "-" && currency != "-" {
		return coin + currency
	}
	return symbol
}

func cryptoOrderNestedField(order service.CryptoOrder, key string, nestedKey string) string {
	values, ok := order[key].(map[string]any)
	if !ok {
		return "-"
	}
	return formatCryptoOrderFieldValue(values[nestedKey])
}

func cryptoFillField(fill service.CryptoFill, keys ...string) string {
	for _, key := range keys {
		value := formatCryptoOrderFieldValue(fill[key])
		if len(value) > 0 && value != "<nil>" {
			return value
		}
	}
	return "-"
}

func cryptoFillSymbol(fill service.CryptoFill) string {
	symbol := cryptoFillField(fill, "symbol", "code", "trade_pair", "trading_pair")
	if symbol != "-" {
		return symbol
	}
	symbol = cryptoFillNestedField(fill, "symbol_pair", "symbol")
	if symbol != "-" {
		return symbol
	}
	coin := cryptoFillField(fill, "coin", "base_currency", "base_coin")
	currency := cryptoFillField(fill, "currency", "quote_currency", "quote_coin")
	if coin == "-" {
		coin = cryptoFillNestedField(fill, "symbol_pair", "base")
	}
	if currency == "-" {
		currency = cryptoFillNestedField(fill, "symbol_pair", "quote")
	}
	if coin != "-" && currency != "-" {
		return coin + currency
	}
	return symbol
}

func cryptoFillNestedField(fill service.CryptoFill, key string, nestedKey string) string {
	values, ok := fill[key].(map[string]any)
	if !ok {
		return "-"
	}
	return formatCryptoOrderFieldValue(values[nestedKey])
}

func formatCryptoFillTime(fill service.CryptoFill) string {
	fillTime := cryptoFillTimestamp(fill)
	if fillTime > 0 {
		return formatTimestamp(fillTime)
	}
	return "-"
}

func cryptoFillTimestamp(fill service.CryptoFill) int64 {
	for _, key := range []string{
		"fill_time",
		"filled_time",
		"deal_time",
		"trade_time",
		"create_time",
		"created_time",
		"time",
		"update_time",
	} {
		if timestamp := parseCryptoTimestamp(fill[key]); timestamp > 0 {
			return timestamp
		}
	}
	return 0
}

func formatCryptoDetailValue(key string, value any) string {
	if isEmptyCryptoDetailValue(key, value) {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any, []any:
		return formatDetailValue("", typed)
	default:
		if strings.HasSuffix(key, "_time") {
			return formatTimestampValue(value)
		}
		return formatCryptoOrderFieldValue(value)
	}
}

func isEmptyCryptoDetailValue(key string, value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return len(strings.TrimSpace(typed)) == 0
	case float64:
		return strings.HasSuffix(key, "_time") && typed == 0
	case int64:
		return strings.HasSuffix(key, "_time") && typed == 0
	case int:
		return strings.HasSuffix(key, "_time") && typed == 0
	default:
		return false
	}
}

func formatCryptoOrderTime(order service.CryptoOrder) string {
	for _, key := range []string{"create_time", "created_time", "time"} {
		if formatted := formatTimestampValue(order[key]); len(formatted) > 0 {
			return formatted
		}
	}
	return "-"
}

func formatCryptoOrderFieldValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case float64:
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), cryptoTimestampParseBase)
		}
		return strconv.FormatFloat(typed, 'f', -1, cryptoTimestampParseBits)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func formatTimestampValue(value any) string {
	timestamp := parseCryptoTimestamp(value)
	if timestamp <= 0 {
		return formatTimestampStringFallback(value)
	}
	return formatTimestamp(timestamp)
}

func parseCryptoTimestamp(value any) int64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), cryptoTimestampParseBits)
		if err != nil {
			return 0
		}
		return int64(parsed)
	}
}

func formatTimestampStringFallback(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func formatTimestampString(value string) string {
	timestamp := parseCryptoTimestamp(value)
	if timestamp <= 0 {
		return strings.TrimSpace(value)
	}
	return formatTimestamp(timestamp)
}

func parseCryptoFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), cryptoTimestampParseBits)
	if err != nil {
		return 0
	}
	return parsed
}

func formatCryptoAveragePrice(summary cryptoFillOrderSummary) string {
	if summary.Qty == 0 {
		return "-"
	}
	return formatCryptoFloat(summary.Amount / summary.Qty)
}

func formatCryptoFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, cryptoTimestampParseBits)
}

func formatCryptoSummaryTime(value int64) string {
	if value <= 0 {
		return "-"
	}
	return formatTimestamp(value)
}

func renderCryptoBalanceTable(balance *service.CryptoTotalBalance) {
	tw := output.NewTableWriter(os.Stdout,
		cryptoBalanceHeaderSection,
		cryptoBalanceHeaderAsset,
		cryptoBalanceHeaderField,
		cryptoBalanceHeaderValue,
	)
	addCryptoBalanceRows(tw, cryptoBalanceSectionCash, balance.CashList)
	addCryptoBalanceRows(tw, cryptoBalanceSectionAsset, balance.Positions)
	tw.Render()
}

func addCryptoBalanceRows(tw *output.TableWriter, section string, rows []service.CryptoBalanceItem) {
	for _, row := range rows {
		asset := cryptoBalanceAsset(row)
		for _, key := range sortedCryptoBalanceKeys(row) {
			tw.AddRow(section, asset, key, fmt.Sprint(row[key]))
		}
	}
}

func cryptoBalanceAsset(row service.CryptoBalanceItem) string {
	for _, key := range []string{"currency", "coin", "symbol", "asset"} {
		if value := strings.TrimSpace(fmt.Sprint(row[key])); len(value) > 0 && value != "<nil>" {
			return value
		}
	}
	return "-"
}

func sortedCryptoBalanceKeys(row service.CryptoBalanceItem) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
