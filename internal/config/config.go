package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Default configuration values.
const (
	DefaultAPIBase    = "https://webapi.futunn.com"
	DefaultTelemetry  = true
	DefaultAutoUpdate = true

	configDirName  = ".futu"
	configFileName = "config.json"
	tokenFileName  = "token.json"

	dirPermissions  = 0o700
	filePermissions = 0o600
)

// Config field key names used by Get/Set.
const (
	KeyAPIBase     = "api_base"
	KeyClientID    = "client_id"
	KeyRedirectURI = "redirect_uri"
	KeyTokenFile   = "token_file"
	KeyDefaultAcct = "default_account"
	KeyTelemetry   = "telemetry"
	KeyAutoUpdate  = "auto_update"
)

// Config holds the CLI configuration.
type Config struct {
	APIBase     string `json:"api_base"`
	ClientID    string `json:"client_id,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`
	TokenFile   string `json:"token_file"`
	DefaultAcct string `json:"default_account"`
	Telemetry   bool   `json:"telemetry"`
	AutoUpdate  bool   `json:"auto_update"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		APIBase:    DefaultAPIBase,
		TokenFile:  filepath.Join(ConfigDir(), tokenFileName),
		Telemetry:  DefaultTelemetry,
		AutoUpdate: DefaultAutoUpdate,
	}
}

// ConfigDir returns the path to the ~/.futu/ directory.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", configDirName)
	}
	return filepath.Join(home, configDirName)
}

// configFilePath returns the full path to the config file.
func configFilePath() string {
	return filepath.Join(ConfigDir(), configFileName)
}

// Load reads configuration from ~/.futu/config.json.
// If the file does not exist, it creates one with default values.
func Load() (*Config, error) {
	path := configFilePath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return createDefault(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return parseConfig(data)
}

// createDefault writes a default config file and returns the default config.
func createDefault(path string) (*Config, error) {
	cfg := DefaultConfig()
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := writeConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseConfig unmarshals JSON data into a Config struct.
func parseConfig(data []byte) (*Config, error) {
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

// Save writes the current configuration to ~/.futu/config.json.
func (c *Config) Save() error {
	path := configFilePath()
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return writeConfig(path, c)
}

// ensureDir creates the directory if it does not exist.
func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}

// writeConfig marshals the config and writes it to the given path.
func writeConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, filePermissions); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// Get returns the config value for the given key as a string.
func (c *Config) Get(key string) string {
	switch key {
	case KeyAPIBase:
		return c.APIBase
	case KeyClientID:
		return c.ClientID
	case KeyRedirectURI:
		return c.RedirectURI
	case KeyTokenFile:
		return c.TokenFile
	case KeyDefaultAcct:
		return c.DefaultAcct
	case KeyTelemetry:
		return strconv.FormatBool(c.Telemetry)
	case KeyAutoUpdate:
		return strconv.FormatBool(c.AutoUpdate)
	default:
		return ""
	}
}

// Set updates the config value for the given key.
func (c *Config) Set(key, value string) error {
	switch key {
	case KeyAPIBase:
		c.APIBase = value
	case KeyClientID:
		c.ClientID = value
	case KeyRedirectURI:
		c.RedirectURI = value
	case KeyTokenFile:
		if len(value) == 0 {
			return fmt.Errorf("token_file cannot be empty")
		}
		c.TokenFile = value
	case KeyDefaultAcct:
		c.DefaultAcct = value
	case KeyTelemetry:
		return c.setBool(&c.Telemetry, key, value)
	case KeyAutoUpdate:
		return c.setBool(&c.AutoUpdate, key, value)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// setBool parses a boolean value and assigns it to the target field.
func (c *Config) setBool(target *bool, key, value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid boolean value for %s: %q", key, value)
	}
	*target = b
	return nil
}
