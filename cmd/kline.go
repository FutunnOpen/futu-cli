package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
	"github.com/futunn/futu/pkg/symbol"
)

const (
	defaultCurKLineNum       = 100
	defaultKLineType         = 2
	defaultCurKLineAUType    = 1
	defaultKLineExtendedTime = 0
	defaultKLinePredSide     = 0
	klineTypeUsage           = "K-line type (default: 2 daily). Seconds: 101=1s 105=5s 110=10s 115=15s 130=30s 145=45s; regular: 1=1m 2=day 3=week 4=month 5=year 6=5m 7=15m 8=30m 9=60m 10=3m 11=quarter 14=120m 15=240m 26=10m 29=180m"
	klineExtendedTimeUsage   = "Extended-hours switch (default: 0). 0=regular only 1=include pre/post-market for US intraday bars 2=include overnight for US"
	klinePredSideUsage       = "Event-contract side (default: 0). 0=keep symbol as-is 1=YES 2=NO"
)

var klineFlags struct {
	num          int
	ktype        int
	autype       int
	extendedTime int
	predSide     int
}

var klineCmd = &cobra.Command{
	Use:     "kline <symbol>",
	GroupID: quoteGroupID,
	Short:   "Get latest K-line data",
	Long:    "Get the latest N K-line bars for the specified symbol. Use history-kline for a historical date range.",
	Example: `  futu kline HK.00700
  futu kline US.AAPL --ktype 1 --extended-time 2
  futu kline HK.00700 --autype 0 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runKLine,
}

func init() {
	klineCmd.Flags().IntVarP(&klineFlags.num, "num", "n", defaultCurKLineNum,
		"Number of K-line bars to return (default: 100, range 1-370)")
	klineCmd.Flags().IntVar(&klineFlags.ktype, "ktype", defaultKLineType, klineTypeUsage)
	klineCmd.Flags().IntVar(&klineFlags.autype, "autype", defaultCurKLineAUType,
		"Adjustment type (default: 1 forward-adjusted without dividends). 0=none 1=forward 2=backward 3=forward with dividends 4=backward with dividends")
	klineCmd.Flags().IntVar(&klineFlags.extendedTime, "extended-time", defaultKLineExtendedTime, klineExtendedTimeUsage)
	klineCmd.Flags().IntVar(&klineFlags.predSide, "pred-side", defaultKLinePredSide, klinePredSideUsage)
	rootCmd.AddCommand(klineCmd)
}

func runKLine(cmd *cobra.Command, args []string) error {
	sym, err := symbol.Parse(args[0])
	if err != nil {
		return fmt.Errorf("parsing symbol: %w", err)
	}
	result, err := quoteSvc.GetCurKLine(cmdContext(cmd), sym.FutuCode(), service.CurKLineParams{
		Num:          klineFlags.num,
		KType:        klineFlags.ktype,
		AUType:       klineFlags.autype,
		ExtendedTime: klineFlags.extendedTime,
		PredSide:     klineFlags.predSide,
	})
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderKLineTable(jsonObjectList(decodeJSONValue(result.Data), "kline_list"), "open_price", "high_price", "low_price", "close_price")
	})
	return nil
}

func renderKLineTable(items []map[string]any, openKey, highKey, lowKey, closeKey string) {
	tw := output.NewTableWriter(os.Stdout, "Time", "Open", "High", "Low", "Close", "Volume")
	for _, item := range items {
		timeValue := jsonInt(item, "time_key")
		if timeValue == 0 {
			timeValue = jsonInt(item, "time")
		}
		tw.AddRow(
			formatTimestamp(timeValue),
			formatPrice(jsonFloat(item, openKey)),
			formatPrice(jsonFloat(item, highKey)),
			formatPrice(jsonFloat(item, lowKey)),
			formatPrice(jsonFloat(item, closeKey)),
			fmt.Sprintf("%d", jsonInt(item, "volume")),
		)
	}
	tw.Render()
}
