package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test network
	if cfg.Network != "emulator" {
		t.Errorf("expected network 'emulator', got '%s'", cfg.Network)
	}

	// Test indexer
	if cfg.Indexer.PollingInterval != 200*time.Millisecond {
		t.Errorf("expected polling interval 200ms, got %v", cfg.Indexer.PollingInterval)
	}

	// Test ports
	if cfg.Ports.Emulator.GRPC != 3569 {
		t.Errorf("expected emulator gRPC port 3569, got %d", cfg.Ports.Emulator.GRPC)
	}

	// Test logging
	if cfg.Logging.Level.Global != "info" {
		t.Errorf("expected global log level 'info', got '%s'", cfg.Logging.Level.Global)
	}

	// Test UI
	if cfg.UI.History.MaxTransactions != 10000 {
		t.Errorf("expected max transactions 10000, got %d", cfg.UI.History.MaxTransactions)
	}
}

func TestLoadWithoutConfigFile(t *testing.T) {
	// Load without any config file - should use defaults
	cfg, err := Load("", zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Network != "emulator" {
		t.Errorf("expected default network 'emulator', got '%s'", cfg.Network)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "aether.yaml")

	configContent := `
network: testnet
indexer:
  polling_interval: 500ms
ports:
  emulator:
    grpc: 4000
logging:
  level:
    global: debug
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config file
	cfg, err := Load(configPath, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	// Verify overrides
	if cfg.Network != "testnet" {
		t.Errorf("expected network 'testnet', got '%s'", cfg.Network)
	}

	if cfg.Indexer.PollingInterval != 500*time.Millisecond {
		t.Errorf("expected polling interval 500ms, got %v", cfg.Indexer.PollingInterval)
	}

	if cfg.Ports.Emulator.GRPC != 4000 {
		t.Errorf("expected emulator gRPC port 4000, got %d", cfg.Ports.Emulator.GRPC)
	}

	if cfg.Logging.Level.Global != "debug" {
		t.Errorf("expected global log level 'debug', got '%s'", cfg.Logging.Level.Global)
	}

	// Verify defaults are still applied for non-overridden values
	if cfg.Ports.DevWallet != 8701 {
		t.Errorf("expected default dev_wallet port 8701, got %d", cfg.Ports.DevWallet)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid network",
			modify: func(c *Config) {
				c.Network = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid port range",
			modify: func(c *Config) {
				c.Ports.Emulator.GRPC = 99999
			},
			wantErr: true,
		},
		{
			name: "port conflict",
			modify: func(c *Config) {
				c.Ports.Emulator.GRPC = 8545
				c.Ports.EVM.RPC = 8545
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level.Global = "invalid"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogLevelInheritance(t *testing.T) {
	cfg := &Config{
		Logging: LoggingConfig{
			Level: LogLevelConfig{
				Global:     "debug",
				Aether:     "",
				Emulator:   "info",
				DevWallet:  "",
				EVMGateway: "",
			},
		},
	}

	applyLogLevelInheritance(cfg)

	// Aether should inherit from global
	if cfg.Logging.Level.Aether != "debug" {
		t.Errorf("expected aether to inherit 'debug', got '%s'", cfg.Logging.Level.Aether)
	}

	// Emulator should keep its explicit value
	if cfg.Logging.Level.Emulator != "info" {
		t.Errorf("expected emulator to keep 'info', got '%s'", cfg.Logging.Level.Emulator)
	}

	// DevWallet should inherit from global
	if cfg.Logging.Level.DevWallet != "debug" {
		t.Errorf("expected dev_wallet to inherit 'debug', got '%s'", cfg.Logging.Level.DevWallet)
	}

	// EVMGateway should default to error
	if cfg.Logging.Level.EVMGateway != "error" {
		t.Errorf("expected evm_gateway to default to 'error', got '%s'", cfg.Logging.Level.EVMGateway)
	}
}
