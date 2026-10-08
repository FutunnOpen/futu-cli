package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/output"
	"github.com/FutunnOpen/futu-cli/internal/service"
)

// Table header labels for the deal command.
const (
	dealHeaderDealID  = "DealID"
	dealHeaderOrderID = "OrderID"
	dealHeaderSymbol  = "Symbol"
	dealHeaderName    = "Name"
	dealHeaderSide    = "Side"
	dealHeaderPrice   = "Price"
	dealHeaderQty     = "Qty"
	dealHeaderStatus  = "Status"
	dealHeaderTime    = "Time"
	dealHeaderUpdated = "Updated"
)

var dealCmd = &cobra.Command{
	Use:   "deal",
	Short: "查询当日成交",
	Long:  "查询指定账户的当日成交记录，支持按交易市场分页获取。",
	RunE:  runDeal,
}

var dealHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "查询历史成交",
	Long:  "查询指定账户的历史成交记录，支持按市场、时间范围和标的过滤。",
	Args:  cobra.NoArgs,
	RunE:  runDealHistory,
}

func init() {
	registerDealFlags(dealCmd)
	registerDealHistoryFlags(dealHistoryCmd)
	dealCmd.AddCommand(dealHistoryCmd)
	rootCmd.AddCommand(dealCmd)
}

func registerDealFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("account", "a", "", "交易账户（默认: 配置 default_account）")
	cmd.Flags().StringP("market", "m", service.DefaultTradingMarket, "交易市场: HK, US, HKCC, SG, JP, AU, MY, KR（默认: HK）")
	cmd.Flags().Int("page-size", service.DefaultPageSize, "每页数量，范围 10-100（默认: 50）")
}

func registerDealHistoryFlags(cmd *cobra.Command) {
	registerDealFlags(cmd)
	cmd.Flags().Int64("start", 0, "查询开始时间，微秒时间戳（默认: 不限制）")
	cmd.Flags().Int64("end", 0, "查询结束时间，微秒时间戳（默认: 不限制）")
	cmd.Flags().String("symbol", "", "标的代码过滤（默认: 不过滤）")
}

// runDeal is the top-level handler for the deal command.
func runDeal(cmd *cobra.Command, _ []string) error {
	req, err := buildTodayDealsRequest(cmd)
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	deals, err := tradeSvc.ListTodayDeals(ctx, req)
	if err != nil {
		return fmt.Errorf("fetching deals: %w", err)
	}

	printResult(deals, func() {
		renderDealTable(deals)
	})
	return nil
}

func runDealHistory(cmd *cobra.Command, _ []string) error {
	req, err := buildHistoryDealsRequest(cmd)
	if err != nil {
		return err
	}
	ctx := cmdContext(cmd)
	deals, err := tradeSvc.ListHistoryDeals(ctx, req)
	if err != nil {
		return fmt.Errorf("fetching history deals: %w", err)
	}
	printResult(deals, func() {
		renderDealTable(deals)
	})
	return nil
}

func buildTodayDealsRequest(cmd *cobra.Command) (service.TodayDealsRequest, error) {
	account, market, pageSize, err := readDealBaseFlags(cmd)
	if err != nil {
		return service.TodayDealsRequest{}, err
	}
	return service.TodayDealsRequest{
		Account:  account,
		Market:   market,
		PageSize: pageSize,
	}, nil
}

func buildHistoryDealsRequest(cmd *cobra.Command) (service.HistoryDealsRequest, error) {
	account, market, pageSize, err := readDealBaseFlags(cmd)
	if err != nil {
		return service.HistoryDealsRequest{}, err
	}
	start, err := cmd.Flags().GetInt64("start")
	if err != nil {
		return service.HistoryDealsRequest{}, fmt.Errorf("reading start flag: %w", err)
	}
	end, err := cmd.Flags().GetInt64("end")
	if err != nil {
		return service.HistoryDealsRequest{}, fmt.Errorf("reading end flag: %w", err)
	}
	symbol, err := cmd.Flags().GetString("symbol")
	if err != nil {
		return service.HistoryDealsRequest{}, fmt.Errorf("reading symbol flag: %w", err)
	}
	return service.HistoryDealsRequest{
		Account:  account,
		Market:   market,
		Start:    start,
		End:      end,
		Symbol:   strings.TrimSpace(symbol),
		PageSize: pageSize,
	}, nil
}

func readDealBaseFlags(cmd *cobra.Command) (string, string, int, error) {
	account, err := cmd.Flags().GetString("account")
	if err != nil {
		return "", "", 0, fmt.Errorf("reading account flag: %w", err)
	}
	account = resolveAccount(account)
	if len(account) == 0 {
		return "", "", 0, fmt.Errorf("account ID is required; use --account or configure default_account")
	}
	market, err := cmd.Flags().GetString("market")
	if err != nil {
		return "", "", 0, fmt.Errorf("reading market flag: %w", err)
	}
	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return "", "", 0, fmt.Errorf("reading page-size flag: %w", err)
	}
	return account, market, pageSize, nil
}

// renderDealTable writes deal data as a formatted table to stdout.
func renderDealTable(deals []service.Deal) {
	tw := output.NewTableWriter(os.Stdout,
		dealHeaderDealID,
		dealHeaderOrderID,
		dealHeaderSymbol,
		dealHeaderName,
		dealHeaderSide,
		dealHeaderPrice,
		dealHeaderQty,
		dealHeaderStatus,
		dealHeaderTime,
		dealHeaderUpdated,
	)

	for _, d := range deals {
		tw.AddRow(
			d.DealID,
			d.OrderID,
			d.Symbol,
			d.StockName,
			d.Side,
			fmt.Sprintf("%.3f", d.Price),
			fmt.Sprintf("%d", d.Qty),
			d.Status,
			formatTimestamp(d.Time),
			formatTimestamp(d.UpdatedTime),
		)
	}
	tw.Render()
}
