package config

import (
	"fmt"
	"strings"
)

// validate validates the configuration
func validate(cfg *Config) error {
	// Validate network mode
	if err := validateNetwork(cfg.Network); err != nil {
		return err
	}

	// Validate ports
	if err := validatePorts(cfg.Ports); err != nil {
		return err
	}

	// Validate log levels
	if err := validateLogLevels(cfg.Logging.Level); err != nil {
		return err
	}

	// Validate UI settings
	if err := validateUI(cfg.UI); err != nil {
		return err
	}

	return nil
}

// validateNetwork validates the network mode
func validateNetwork(network string) error {
	validNetworks := map[string]bool{
		"emulator": true,
		"testnet":  true,
		"mainnet":  true,
	}

	if !validNetworks[network] {
		return fmt.Errorf("invalid network mode '%s': must be one of: emulator, testnet, mainnet", network)
	}

	return nil
}

// validatePorts validates all port configurations
func validatePorts(ports PortsConfig) error {
	// Collect all ports to check for conflicts
	portMap := make(map[int]string)

	checkPort := func(port int, name string) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %d for %s: must be between 1 and 65535", port, name)
		}
		if existing, exists := portMap[port]; exists {
			return fmt.Errorf("port conflict: %d is used by both %s and %s", port, existing, name)
		}
		portMap[port] = name
		return nil
	}

	// Check emulator ports
	if err := checkPort(ports.Emulator.GRPC, "emulator.grpc"); err != nil {
		return err
	}
	if err := checkPort(ports.Emulator.REST, "emulator.rest"); err != nil {
		return err
	}
	if err := checkPort(ports.Emulator.Admin, "emulator.admin"); err != nil {
		return err
	}
	if err := checkPort(ports.Emulator.Debugger, "emulator.debugger"); err != nil {
		return err
	}

	// Check other ports
	if err := checkPort(ports.DevWallet, "dev_wallet"); err != nil {
		return err
	}
	if err := checkPort(ports.EVM.RPC, "evm.rpc"); err != nil {
		return err
	}
	if err := checkPort(ports.EVM.Profiler, "evm.profiler"); err != nil {
		return err
	}
	if err := checkPort(ports.EVM.Metrics, "evm.metrics"); err != nil {
		return err
	}

	return nil
}

// validateLogLevels validates log level settings
func validateLogLevels(levels LogLevelConfig) error {
	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	checkLevel := func(level, component string) error {
		if level == "" {
			return nil // Empty is valid (inherits)
		}
		levelLower := strings.ToLower(level)
		if !validLevels[levelLower] {
			return fmt.Errorf("invalid log level '%s' for %s: must be one of: trace, debug, info, warn, error, fatal", level, component)
		}
		return nil
	}

	if err := checkLevel(levels.Global, "global"); err != nil {
		return err
	}
	if err := checkLevel(levels.Aether, "aether"); err != nil {
		return err
	}
	if err := checkLevel(levels.Emulator, "emulator"); err != nil {
		return err
	}
	if err := checkLevel(levels.DevWallet, "dev_wallet"); err != nil {
		return err
	}
	if err := checkLevel(levels.EVMGateway, "evm_gateway"); err != nil {
		return err
	}

	return nil
}

// validateUI validates UI configuration
func validateUI(ui UIConfig) error {
	// Validate percentages
	if ui.Layout.TransactionsSplitPercent < 0 || ui.Layout.TransactionsSplitPercent > 100 {
		return fmt.Errorf("invalid transactions split percent: must be between 0 and 100")
	}
	if ui.Layout.EventsSplitPercent < 0 || ui.Layout.EventsSplitPercent > 100 {
		return fmt.Errorf("invalid events split percent: must be between 0 and 100")
	}
	if ui.Layout.RunnerSplitPercent < 0 || ui.Layout.RunnerSplitPercent > 100 {
		return fmt.Errorf("invalid runner split percent: must be between 0 and 100")
	}

	// Validate positive values
	if ui.History.MaxTransactions < 1 {
		return fmt.Errorf("max_transactions must be at least 1")
	}
	if ui.History.MaxEvents < 1 {
		return fmt.Errorf("max_events must be at least 1")
	}

	return nil
}
