package config

import "time"

// DefaultConfig returns a Config with sensible defaults matching current hardcoded behavior
func DefaultConfig() *Config {
	return &Config{
		Network: "emulator",
		Flow: FlowConfig{
			NewUserBalance: 1000.0,
			BlockTime:      1 * time.Second,
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
			Theme: "solarized",
			History: HistoryConfig{
				MaxTransactions: 10000,
				MaxEvents:       10000,
				MaxLogLines:     10000,
			},
			Layout: LayoutConfig{
				DefaultView: "transactions",
				Transactions: ViewLayoutConfig{
					TableWidthPercent:  40,
					DetailWidthPercent: 60,
				},
				Events: ViewLayoutConfig{
					TableWidthPercent:  60,
					DetailWidthPercent: 40,
				},
			},
			Defaults: DefaultsConfig{
				ShowEventFields:  false,
				ShowRawAddresses: false,
				FullDetailMode:   false,
			},
			Filter: FilterConfig{
				CharLimit: 50,
				Width:     50,
			},
			Save: SaveConfig{
				DefaultDirectory:  "transactions",
				FilenameCharLimit: 50,
				DialogWidth:       40,
			},
		},
	}
}
