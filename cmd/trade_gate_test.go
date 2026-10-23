package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/dryrun"
)

func TestGateTradeExecutionDefaultsToDryRun(t *testing.T) {
	previous := outputFormat
	t.Cleanup(func() { outputFormat = previous })
	outputFormat = formatJSON

	scope := dryrun.Order("BUY", "US.AAPL", "1", "100")
	output := captureStdout(t, func() {
		proceed, err := gateTradeExecution(scope, "", map[string]any{
			"side":   "BUY",
			"symbol": "US.AAPL",
			"qty":    int64(1),
			"price":  "100",
		})
		if err != nil {
			t.Fatalf("gateTradeExecution() error = %v", err)
		}
		if proceed {
			t.Fatal("expected dry-run to block submission")
		}
	})
	if !strings.Contains(output, `"dry_run"`) || !strings.Contains(output, "true") {
		t.Fatalf("output = %s", output)
	}
	if !strings.Contains(output, scope.Code()) {
		t.Fatalf("missing confirmation code in output: %s", output)
	}
}

func TestGateTradeExecutionRequiresMatchingCode(t *testing.T) {
	scope := dryrun.Order("BUY", "US.AAPL", "1", "100")
	proceed, err := gateTradeExecution(scope, "000", map[string]any{"side": "BUY"})
	if err == nil || proceed {
		t.Fatalf("expected mismatch error, proceed=%v err=%v", proceed, err)
	}
	proceed, err = gateTradeExecution(scope, scope.Code(), map[string]any{"side": "BUY"})
	if err != nil || !proceed {
		t.Fatalf("expected successful gate, proceed=%v err=%v", proceed, err)
	}
}

func TestMutatingOrderCommandsExposeExecuteFlag(t *testing.T) {
	commands := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "buy", cmd: orderBuyCmd},
		{name: "sell", cmd: orderSellCmd},
		{name: "place", cmd: orderPlaceCmd},
		{name: "modify", cmd: orderModifyCmd},
		{name: "cancel", cmd: orderCancelCmd},
		{name: "crypto place", cmd: cryptoOrderPlaceCmd},
		{name: "crypto buy", cmd: cryptoOrderBuyCmd},
		{name: "crypto sell", cmd: cryptoOrderSellCmd},
		{name: "crypto modify", cmd: cryptoOrderModifyCmd},
		{name: "crypto cancel", cmd: cryptoOrderCancelCmd},
	}
	for _, item := range commands {
		if item.cmd.Flags().Lookup(executeFlagName) == nil {
			t.Fatalf("%s missing --%s flag", item.name, executeFlagName)
		}
	}
}
