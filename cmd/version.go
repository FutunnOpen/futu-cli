package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version output format template.
const versionTemplate = "futu version %s (%s/%s)"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Long:  "Print the current version of the Futu CLI tool.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(formatVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// formatVersion returns the version string with OS/arch info.
func formatVersion() string {
	return fmt.Sprintf(versionTemplate, Version, runtime.GOOS, runtime.GOARCH)
}
