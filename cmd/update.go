package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/update"
)

// Update result messages.
const (
	msgNoUpdate       = "You are already on the latest version (%s)."
	msgUpdateAvail    = "Update available: %s -> %s"
	msgUpdateApplied  = "Successfully updated to version %s."
	msgUpdateCheckRun = "Run 'futu update' to apply."
)

var updateCheckOnly bool
var updateForce bool
var updateReleaseNotes bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and apply CLI updates",
	Long:  "Check for a newer version of the Futu CLI. Without --check, downloads and applies the update.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false,
		"Only check for updates without applying (default: false)")
	updateCmd.Flags().BoolVar(&updateForce, "force", false,
		"Reinstall the latest version even if already installed (default: false)")
	updateCmd.Flags().BoolVar(&updateReleaseNotes, "release-notes", false,
		"Show release notes without applying an update (default: false)")
	rootCmd.AddCommand(updateCmd)
}

// runUpdate checks for updates and optionally applies them.
func runUpdate() error {
	if updateReleaseNotes {
		return printReleaseNotes()
	}

	info, err := update.Check(Version)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	if !info.Available && !updateForce {
		fmt.Printf(msgNoUpdate+"\n", Version)
		return nil
	}
	if updateForce {
		info.Available = true
	}

	printUpdateInfo(info)

	if updateCheckOnly {
		fmt.Println(msgUpdateCheckRun)
		return nil
	}

	return applyUpdate(info)
}

func printReleaseNotes() error {
	notes, err := update.ReleaseNotes()
	if err != nil {
		return fmt.Errorf("fetch release notes: %w", err)
	}
	fmt.Println(notes)
	return nil
}

// printUpdateInfo displays the available update details.
func printUpdateInfo(info *update.UpdateInfo) {
	fmt.Printf(msgUpdateAvail+"\n", info.CurrentVersion, info.LatestVersion)
	if len(info.ReleaseNotes) > 0 {
		fmt.Printf("\nRelease notes:\n%s\n\n", info.ReleaseNotes)
	}
}

// applyUpdate downloads and installs the update.
func applyUpdate(info *update.UpdateInfo) error {
	if err := update.Apply(info); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}
	fmt.Printf(msgUpdateApplied+"\n", info.LatestVersion)
	return nil
}
