package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Load loads configuration from file with the following priority:
// 1. Explicit path via configPath parameter
// 2. ./aether.yaml (current directory)
// 3. ./config/aether.yaml
// 4. ~/.aether/config.yaml (user home)
// 5. /etc/aether/config.yaml (system-wide)
// Falls back to defaults if no config file is found
func Load(configPath string, logger zerolog.Logger) (*Config, error) {
	v := viper.New()

	// Configure viper
	v.SetConfigName("aether")
	v.SetConfigType("yaml")

	// Add search paths
	if configPath != "" {
		// Explicit path provided
		v.SetConfigFile(configPath)
	} else {
		// Add default search paths
		v.AddConfigPath(".")        // Current directory
		v.AddConfigPath("./config") // ./config directory
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
	configFileUsed := ""
	foundConfigFile := false
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults
			// This is not an error - we have sensible defaults
			logger.Debug().
				Str("searchPaths", "., ./config, ~/.aether, /etc/aether").
				Msg("No config file found in search paths, using defaults from defaults.go")
		} else {
			// Config file was found but another error occurred
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		configFileUsed = v.ConfigFileUsed()
		foundConfigFile = true
		logger.Debug().
			Str("configFile", configFileUsed).
			Msg("Config file found and loaded by viper")
	}

	// Start with defaults from defaults.go
	cfg := DefaultConfig()
	logger.Debug().Msg("Starting with defaults from defaults.go")
	
	// Unmarshal config file on top of defaults (if file exists)
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Always log complete effective configuration
	logger.Info().
		Bool("configFileFound", foundConfigFile).
		Str("configFile", configFileUsed).
		Str("network", cfg.Network).
		Interface("flow", cfg.Flow).
		Interface("indexer", cfg.Indexer).
		Interface("ports", cfg.Ports).
		Interface("evm", cfg.EVM).
		Interface("logging", cfg.Logging).
		Interface("ui", cfg.UI).
		Msg("Complete effective configuration")

	// Apply inheritance for log levels
	applyLogLevelInheritance(cfg)

	// Validate configuration
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
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
	// EVMGateway defaults to "error" instead of inheriting from global
	if cfg.Logging.Level.EVMGateway == "" {
		cfg.Logging.Level.EVMGateway = "error"
	}
}
