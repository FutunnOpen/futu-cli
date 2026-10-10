package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
)

const (
	defaultHistoryKLineNum    = 100
	defaultHistoryKLineAUType = 1
)

var historyKLineFlags struct {
	start        string
	end          string
	nextTime     int64
	ktype        int
	autype       int
	num          int
	extendedTime int
	predSide     int
}

var historyKLineCmd = &cobra.Command{
	Use:     "history-kline <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get historical K-line data",
	Long:    "Get historical K-line bars for the specified range. For pagination, pass the previous response next_time to --next-time.",
	Example: `  futu history-kline HK.00700
  futu history-kline US.AAPL --start 2026-01-01 --end 2026-03-01 --num 100
  futu history-kline HK.00700 --next-time 1779163200000 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runHistoryKLine,
}

func init() {
	historyKLineCmd.Flags().StringVar(&historyKLineFlags.start, "start", "",
		"Start date yyyy-MM-dd, inclusive (default: inferred backward from end by num)")
	historyKLineCmd.Flags().StringVar(&historyKLineFlags.end, "end", "",
		"End date yyyy-MM-dd, inclusive (default: today)")
	historyKLineCmd.Flags().Int64Var(&historyKLineFlags.nextTime, "next-time", 0,
		"Backward pagination cursor as a millisecond timestamp (default: 0, disabled). Pass the previous response next_time; takes precedence over --end")
	historyKLineCmd.Flags().IntVar(&historyKLineFlags.ktype, "ktype", defaultKLineType, klineTypeUsage)
	historyKLineCmd.Flags().IntVar(&historyKLineFlags.autype, "autype", defaultHistoryKLineAUType,
		"Adjustment type (default: 1 forward-adjusted). 0=none 1=forward 2=backward")
	historyKLineCmd.Flags().IntVarP(&historyKLineFlags.num, "num", "n", defaultHistoryKLineNum,
		"Number of K-line bars to return (default: 100, max 370)")
	historyKLineCmd.Flags().IntVar(&historyKLineFlags.extendedTime, "extended-time", defaultKLineExtendedTime, klineExtendedTimeUsage)
	historyKLineCmd.Flags().IntVar(&historyKLineFlags.predSide, "pred-side", defaultKLinePredSide, klinePredSideUsage)
	rootCmd.AddCommand(historyKLineCmd)
}

func runHistoryKLine(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}
	end := historyKLineFlags.end
	if len(end) == 0 && historyKLineFlags.nextTime == 0 {
		end = time.Now().Format("2006-01-02")
	}
	result, err := quoteSvc.GetHistoryKLine(cmdContext(cmd), sym.FutuCode(), service.HistoryKLineParams{
		Start:        historyKLineFlags.start,
		End:          end,
		NextTime:     historyKLineFlags.nextTime,
		KType:        historyKLineFlags.ktype,
		AUType:       historyKLineFlags.autype,
		Num:          historyKLineFlags.num,
		ExtendedTime: historyKLineFlags.extendedTime,
		PredSide:     historyKLineFlags.predSide,
	})
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderKLineTable(jsonObjectList(decodeJSONValue(result.Data), "kline_list"), "open", "high", "low", "close")
	})
	return nil
}
