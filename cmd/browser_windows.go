package cmd

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func openBrowser(targetURL string) error {
	url, err := windows.UTF16PtrFromString(targetURL)
	if err != nil {
		return fmt.Errorf("encode browser URL: %w", err)
	}
	if err := windows.ShellExecute(0, nil, url, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
