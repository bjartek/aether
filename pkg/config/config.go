package config

import (
	"time"
)

// Config represents the complete Aether configuration
type Config struct {
	Network string          `mapstructure:"network"`
	Flow    FlowConfig      `mapstructure:"flow"`
	Indexer IndexerConfig   `mapstructure:"indexer"`
	Ports   PortsConfig     `mapstructure:"ports"`
	EVM     EVMConfig       `mapstructure:"evm"`
	Logging LoggingConfig   `mapstructure:"logging"`
	UI      UIConfig        `mapstructure:"ui"`
}

// FlowConfig contains Flow blockchain settings
type FlowConfig struct {
	NewUserBalance float64       `mapstructure:"new_user_balance"`
	BlockTime      time.Duration `mapstructure:"block_time"`
}

// IndexerConfig contains indexer-specific settings
type IndexerConfig struct {
	PollingInterval time.Duration   `mapstructure:"polling_interval"`
	Underflow       UnderflowConfig `mapstructure:"underflow"`
}

// UnderflowConfig contains data formatting options
type UnderflowConfig struct {
	ByteArrayAsHex       bool   `mapstructure:"byte_array_as_hex"`
	ShowTimestampsAsDate bool   `mapstructure:"show_timestamps_as_date"`
	TimestampFormat      string `mapstructure:"timestamp_format"`
}

// PortsConfig contains all port configurations
type PortsConfig struct {
	Emulator  EmulatorPortsConfig `mapstructure:"emulator"`
	DevWallet int                 `mapstructure:"dev_wallet"`
	EVM       EVMPortsConfig      `mapstructure:"evm"`
}

// EmulatorPortsConfig contains emulator-specific ports
type EmulatorPortsConfig struct {
	GRPC     int `mapstructure:"grpc"`
	REST     int `mapstructure:"rest"`
	Admin    int `mapstructure:"admin"`
	Debugger int `mapstructure:"debugger"`
}

// EVMPortsConfig contains EVM-specific ports
type EVMPortsConfig struct {
	RPC      int `mapstructure:"rpc"`
	Profiler int `mapstructure:"profiler"`
	Metrics  int `mapstructure:"metrics"`
}

// EVMConfig contains EVM gateway settings
type EVMConfig struct {
	DatabasePath          string `mapstructure:"database_path"`
	DeleteDatabaseOnStart bool   `mapstructure:"delete_database_on_start"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level           LogLevelConfig `mapstructure:"level"`
	TimestampFormat string         `mapstructure:"timestamp_format"`
	Color           bool           `mapstructure:"color"`
	File            LogFileConfig  `mapstructure:"file"`
}

// LogLevelConfig contains log levels for each component
type LogLevelConfig struct {
	Global     string `mapstructure:"global"`
	Aether     string `mapstructure:"aether"`
	Emulator   string `mapstructure:"emulator"`
	DevWallet  string `mapstructure:"dev_wallet"`
	EVMGateway string `mapstructure:"evm_gateway"`
}

// LogFileConfig contains file logging settings
type LogFileConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Path       string `mapstructure:"path"`
	BufferSize int    `mapstructure:"buffer_size"`
}

// UIConfig contains UI preferences
type UIConfig struct {
	History  HistoryConfig  `mapstructure:"history"`
	Layout   LayoutConfig   `mapstructure:"layout"`
	Defaults DefaultsConfig `mapstructure:"defaults"`
}

// HistoryConfig contains history limits
type HistoryConfig struct {
	MaxTransactions int `mapstructure:"max_transactions"`
	MaxEvents       int `mapstructure:"max_events"`
	MaxLogLines     int `mapstructure:"max_log_lines"`
}

// LayoutConfig contains layout preferences
type LayoutConfig struct {
	TransactionsSplitPercent int `mapstructure:"transactions_split_percent"` // Table width as percentage (0-100)
	EventsSplitPercent       int `mapstructure:"events_split_percent"`       // Table width as percentage (0-100)
	RunnerSplitPercent       int `mapstructure:"runner_split_percent"`       // Table width as percentage (0-100)
}

// DefaultsConfig contains default display modes
type DefaultsConfig struct {
	ShowEventFields  bool   `mapstructure:"show_event_fields"`
	ShowRawAddresses bool   `mapstructure:"show_raw_addresses"`
	TimeFormat       string `mapstructure:"time_format"` // Go time format string, default "15:04:05"
}
