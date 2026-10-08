package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/config"
	"github.com/FutunnOpen/futu-cli/internal/output"
)

var allConfigKeys = []string{
	config.KeyAPIBase,
	config.KeyClientID,
	config.KeyRedirectURI,
	config.KeyTokenFile,
	config.KeyDefaultAcct,
	config.KeyTelemetry,
	config.KeyAutoUpdate,
}

const (
	configHeaderKey   = "Key"
	configHeaderValue = "Value"
)

var configCmd = &cobra.Command{Use: "config", Short: "管理 CLI 配置"}

var configSetCmd = &cobra.Command{
	Use: "set <key> <value>", Short: "设置配置项",
	Args: cobra.ExactArgs(2), RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use: "get <key>", Short: "查看配置项",
	Args: cobra.ExactArgs(1), RunE: runConfigGet,
}

var configListCmd = &cobra.Command{Use: "list", Short: "列出所有配置项", RunE: runConfigList}
var configInitCmd = &cobra.Command{Use: "init", Short: "初始化并登录 OAuth2", RunE: runConfigInit}

func init() {
	configCmd.AddCommand(configSetCmd, configGetCmd, configListCmd, configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	if err := cfg.Set(key, value); err != nil {
		return fmt.Errorf("set config: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Config %s set to %s\n", key, value)
	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stdout, cfg.Get(args[0]))
	return nil
}

func runConfigList(_ *cobra.Command, _ []string) error {
	data := buildConfigMap()
	printResult(data, renderConfigTable)
	return nil
}

func buildConfigMap() map[string]string {
	values := make(map[string]string, len(allConfigKeys))
	for _, key := range allConfigKeys {
		values[key] = cfg.Get(key)
	}
	return values
}

func renderConfigTable() {
	tw := output.NewTableWriter(os.Stdout, configHeaderKey, configHeaderValue)
	for _, key := range allConfigKeys {
		tw.AddRow(key, cfg.Get(key))
	}
	tw.Render()
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stderr, "正在启动 OAuth2 登录...")
	return runLogin(loginCmd, nil)
}
