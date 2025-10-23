package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Load loads configuration from file with the following priority:
// 1. Explicit path via configPath parameter
// 2. ./aether.yaml (current directory)
// 3. ./config/aether.yaml
// 4. ~/.aether/config.yaml (user home)
// 5. /etc/aether/config.yaml (system-wide)
// Falls back to defaults if no config file is found
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("aether")
	v.SetConfigType("yaml")

	// Add search paths
	if configPath != "" {
		// Explicit path provided
		v.SetConfigFile(configPath)
	} else {
		// Add default search paths
		v.AddConfigPath(".")                    // Current directory
		v.AddConfigPath("./config")             // ./config directory
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".aether")) // ~/.aether
		}
		v.AddConfigPath("/etc/aether") // System-wide
	}

	// Enable environment variable overrides
	// Environment variables use AETHER_ prefix and underscore separators
	// Example: AETHER_NETWORK=testnet, AETHER_LOGGING_LEVEL_GLOBAL=debug
	v.SetEnvPrefix("AETHER")
	v.AutomaticEnv()

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults
			// This is not an error - we have sensible defaults
		} else {
			// Config file was found but another error occurred
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply inheritance for log levels
	applyLogLevelInheritance(cfg)

	// Validate configuration
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	// Network
	v.SetDefault("network", defaults.Network)

	// Flow
	v.SetDefault("flow.new_user_balance", defaults.Flow.NewUserBalance)
	v.SetDefault("flow.block_time", defaults.Flow.BlockTime)

	// Indexer
	v.SetDefault("indexer.polling_interval", defaults.Indexer.PollingInterval)
	v.SetDefault("indexer.underflow.byte_array_as_hex", defaults.Indexer.Underflow.ByteArrayAsHex)
	v.SetDefault("indexer.underflow.show_timestamps_as_date", defaults.Indexer.Underflow.ShowTimestampsAsDate)
	v.SetDefault("indexer.underflow.timestamp_format", defaults.Indexer.Underflow.TimestampFormat)

	// Ports
	v.SetDefault("ports.emulator.grpc", defaults.Ports.Emulator.GRPC)
	v.SetDefault("ports.emulator.rest", defaults.Ports.Emulator.REST)
	v.SetDefault("ports.emulator.admin", defaults.Ports.Emulator.Admin)
	v.SetDefault("ports.emulator.debugger", defaults.Ports.Emulator.Debugger)
	v.SetDefault("ports.dev_wallet", defaults.Ports.DevWallet)
	v.SetDefault("ports.evm.rpc", defaults.Ports.EVM.RPC)
	v.SetDefault("ports.evm.profiler", defaults.Ports.EVM.Profiler)
	v.SetDefault("ports.evm.metrics", defaults.Ports.EVM.Metrics)

	// EVM
	v.SetDefault("evm.database_path", defaults.EVM.DatabasePath)
	v.SetDefault("evm.delete_database_on_start", defaults.EVM.DeleteDatabaseOnStart)

	// Logging
	v.SetDefault("logging.level.global", defaults.Logging.Level.Global)
	v.SetDefault("logging.level.aether", defaults.Logging.Level.Aether)
	v.SetDefault("logging.level.emulator", defaults.Logging.Level.Emulator)
	v.SetDefault("logging.level.dev_wallet", defaults.Logging.Level.DevWallet)
	v.SetDefault("logging.level.evm_gateway", defaults.Logging.Level.EVMGateway)
	v.SetDefault("logging.timestamp_format", defaults.Logging.TimestampFormat)
	v.SetDefault("logging.color", defaults.Logging.Color)
	v.SetDefault("logging.file.enabled", defaults.Logging.File.Enabled)
	v.SetDefault("logging.file.path", defaults.Logging.File.Path)
	v.SetDefault("logging.file.buffer_size", defaults.Logging.File.BufferSize)

	// UI
	v.SetDefault("ui.theme", defaults.UI.Theme)
	v.SetDefault("ui.history.max_transactions", defaults.UI.History.MaxTransactions)
	v.SetDefault("ui.history.max_events", defaults.UI.History.MaxEvents)
	v.SetDefault("ui.history.max_log_lines", defaults.UI.History.MaxLogLines)
	v.SetDefault("ui.layout.default_view", defaults.UI.Layout.DefaultView)
	v.SetDefault("ui.layout.transactions.table_width_percent", defaults.UI.Layout.Transactions.TableWidthPercent)
	v.SetDefault("ui.layout.transactions.detail_width_percent", defaults.UI.Layout.Transactions.DetailWidthPercent)
	v.SetDefault("ui.layout.events.table_width_percent", defaults.UI.Layout.Events.TableWidthPercent)
	v.SetDefault("ui.layout.events.detail_width_percent", defaults.UI.Layout.Events.DetailWidthPercent)
	v.SetDefault("ui.defaults.show_event_fields", defaults.UI.Defaults.ShowEventFields)
	v.SetDefault("ui.defaults.show_raw_addresses", defaults.UI.Defaults.ShowRawAddresses)
	v.SetDefault("ui.defaults.full_detail_mode", defaults.UI.Defaults.FullDetailMode)
	v.SetDefault("ui.defaults.time_format", defaults.UI.Defaults.TimeFormat)
	v.SetDefault("ui.filter.char_limit", defaults.UI.Filter.CharLimit)
	v.SetDefault("ui.filter.width", defaults.UI.Filter.Width)
	v.SetDefault("ui.save.default_directory", defaults.UI.Save.DefaultDirectory)
	v.SetDefault("ui.save.filename_char_limit", defaults.UI.Save.FilenameCharLimit)
	v.SetDefault("ui.save.dialog_width", defaults.UI.Save.DialogWidth)
}

// applyLogLevelInheritance applies log level inheritance
// If a component-specific level is empty, it inherits from global
func applyLogLevelInheritance(cfg *Config) {
	if cfg.Logging.Level.Aether == "" {
		cfg.Logging.Level.Aether = cfg.Logging.Level.Global
	}
	if cfg.Logging.Level.Emulator == "" {
		cfg.Logging.Level.Emulator = cfg.Logging.Level.Global
	}
	if cfg.Logging.Level.DevWallet == "" {
		cfg.Logging.Level.DevWallet = cfg.Logging.Level.Global
	}
	// EVMGateway has its own default (error), so only inherit if explicitly empty
	if cfg.Logging.Level.EVMGateway == "" {
		cfg.Logging.Level.EVMGateway = "error"
	}
}
