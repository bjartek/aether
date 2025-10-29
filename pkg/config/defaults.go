package config

import "time"

// DefaultConfig returns a Config with sensible defaults matching current hardcoded behavior
func DefaultConfig() *Config {
	return &Config{
		Network: "emulator",
		Flow: FlowConfig{
			NewUserBalance: 1000.0,

			//I have not tweaked this on emulator along with block_time
			BlockTime:                   1 * time.Second,
			InitTransactionsFolder:      "",    // Empty string means use root aether folder (no subfolder filtering)
			InitTransactionsInteractive: false, // If true, prompt user to select folder at startup
		},
		Indexer: IndexerConfig{
			//I have not tweaked this in indexer along with block_time
			PollingInterval: 200 * time.Millisecond,
			//these settings affect how things are INDEXED, so if you tweak these eventual filters might need changing
			Underflow: UnderflowConfig{
				ByteArrayAsHex:       true, //i find hex values easier to read, but for absolute correctness set to false
				ShowTimestampsAsDate: true, //some ppl can read unix_timestamps, i cant. so i made this to make it easier on the eyes
				TimestampFormat:      "2006-01-02 15:04:05 UTC",
			},
		},
		//these are here just incase you happen to have bound these ports to something else and still want to run this
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
		//some people might want to put this elsewhere or keep it
		EVM: EVMConfig{
			DatabasePath:          "evm-gateway-db",
			DeleteDatabaseOnStart: true,
		},

		//you never know how people want to log, the evm gateway is very verbose so set it to error
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
				//as standard we do not show a log file in a file for the log of your application, but you can if you need it
				//note that this is not the same as the log to debug aether, this is for your applications joined logs
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
			//if you use a narrower terminal then you could set these
			Layout: LayoutConfig{
				TransactionsSplitPercent: 50,
				EventsSplitPercent:       50,
				RunnerSplitPercent:       40,
			},
			Defaults: DefaultsConfig{
				ShowEventFields:  true,
				ShowRawAddresses: false,
				TimeFormat:       "15:04:05",
				Sort:             "desc", // Show oldest first
			},
		},
		//set this only if you want to run the frontned as part of aether, totally optional
		FrontendCommand: "",
	}
}
