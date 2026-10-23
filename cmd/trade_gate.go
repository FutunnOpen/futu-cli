package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/dryrun"
	"github.com/futunn/futu/internal/output"
)

const (
	dryRunMessage   = "DRY RUN - no order was submitted. Re-run with --execute <confirmation_code> to submit."
	executeFlagName = "execute"
)

type dryRunResult struct {
	DryRun           bool           `json:"dry_run"`
	Submitted        bool           `json:"submitted"`
	ConfirmationCode string         `json:"confirmation_code"`
	Message          string         `json:"message"`
	Preview          map[string]any `json:"preview"`
}

func registerExecuteFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, executeFlagName, "", "Confirmation code from a dry-run preview; required to submit the order")
}

func gateTradeExecution(scope dryrun.Scope, execute string, preview map[string]any) (bool, error) {
	if len(strings.TrimSpace(execute)) == 0 {
		printDryRun(scope.Code(), preview)
		return false, nil
	}
	if err := scope.Verify(execute); err != nil {
		return false, err
	}
	return true, nil
}

func printDryRun(code string, preview map[string]any) {
	result := dryRunResult{
		DryRun:           true,
		Submitted:        false,
		ConfirmationCode: code,
		Message:          dryRunMessage,
		Preview:          preview,
	}
	if isJSONOutput() {
		_ = output.PrintJSON(os.Stdout, result)
		return
	}
	fmt.Fprintln(os.Stderr, "\nDRY RUN - no order was submitted.")
	keys := sortedPreviewKeys(preview)
	for _, key := range keys {
		fmt.Fprintf(os.Stderr, "  %s: %v\n", key, preview[key])
	}
	fmt.Fprintf(os.Stderr, "\nConfirmation code: %s\n", code)
	fmt.Fprintf(os.Stderr, "Re-run the same command with --execute %s to submit.\n\n", code)
}

func sortedPreviewKeys(preview map[string]any) []string {
	keys := make([]string, 0, len(preview))
	for key := range preview {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatQty(value int64) string {
	return strconv.FormatInt(value, 10)
}
