package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
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
	Short: "Query today's fills",
	Long:  "Query today's fills for the specified account, with pagination by trading market.",
	RunE:  runDeal,
}

var dealHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Query historical fills",
	Long:  "Query historical fills for the specified account, with filters for market, time range, and symbol.",
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
	cmd.Flags().StringP("account", "a", "", "Trading account ID (default: config default_account)")
	cmd.Flags().StringP("market", "m", service.DefaultTradingMarket, "Trading market: HK, US, HKCC, SG, JP, AU, MY, KR (default: HK)")
	cmd.Flags().Int("page-size", service.DefaultPageSize, "Page size, range 10-100 (default: 50)")
}

func registerDealHistoryFlags(cmd *cobra.Command) {
	registerDealFlags(cmd)
	cmd.Flags().Int64("start", 0, "Start time as a microsecond timestamp (default: no lower bound)")
	cmd.Flags().Int64("end", 0, "End time as a microsecond timestamp (default: no upper bound)")
	cmd.Flags().String("symbol", "", "Symbol filter (default: no filter)")
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
