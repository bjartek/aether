package config

import "time"

// DefaultConfig returns a Config with sensible defaults matching current hardcoded behavior
func DefaultConfig() *Config {
	return &Config{
		Network: "emulator",
		Flow: FlowConfig{
			NewUserBalance:              1000.0,
			BlockTime:                   1 * time.Second,
			InitTransactionsFolder:      "",    // Empty string means use root aether folder (no subfolder filtering)
			InitTransactionsInteractive: false, // If true, prompt user to select folder at startup
		},
		Indexer: IndexerConfig{
			PollingInterval: 200 * time.Millisecond,
			Underflow: UnderflowConfig{
				ByteArrayAsHex:       true,
				ShowTimestampsAsDate: true,
				TimestampFormat:      "2006-01-02 15:04:05 UTC",
			},
		},
		Ports: PortsConfig{
			Emulator: EmulatorPortsConfig{
				GRPC:     3569,
				REST:     8888,
				Admin:    8080,
				Debugger: 2345,
			},
			DevWallet: 8701,
			EVM: EVMPortsConfig{
				RPC:      8545,
				Profiler: 6060,
				Metrics:  9091,
			},
		},
		EVM: EVMConfig{
			DatabasePath:          "evm-gateway-db",
			DeleteDatabaseOnStart: true,
		},
		Logging: LoggingConfig{
			Level: LogLevelConfig{
				Global:     "info",
				Aether:     "", // inherits global
				Emulator:   "", // inherits global
				DevWallet:  "", // inherits global
				EVMGateway: "error",
			},
			TimestampFormat: "15:04:05",
			Color:           true,
			File: LogFileConfig{
				Enabled:    false,
				Path:       "",
				BufferSize: 1000,
			},
		},
		UI: UIConfig{
			History: HistoryConfig{
				MaxTransactions: 10000,
				MaxEvents:       10000,
				MaxLogLines:     10000,
			},
			Layout: LayoutConfig{
				TransactionsSplitPercent: 50,
				EventsSplitPercent:       50,
				RunnerSplitPercent:       40,
			},
			Defaults: DefaultsConfig{
				ShowEventFields:  true,
				ShowRawAddresses: false,
				TimeFormat:       "15:04:05",
				Sort:             "asc", // Show newest first
			},
		},
		FrontendCommand: "npm start",
	}
}
