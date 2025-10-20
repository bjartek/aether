# Aether Configuration Package

This package provides a flexible, file-based configuration system for Aether using [Viper](https://github.com/spf13/viper).

## Features

- **YAML/JSON Support**: Primary format is YAML for readability, with JSON fallback
- **Sensible Defaults**: All settings have defaults matching current hardcoded behavior
- **Environment Variables**: Override any setting with `AETHER_` prefixed environment variables
- **Validation**: Comprehensive validation including port conflict detection
- **Log Level Inheritance**: Component-specific log levels inherit from global if not set
- **Multiple Search Paths**: Automatic discovery from multiple locations

## Usage

### Basic Usage

```go
import "github.com/bjartek/aether/pkg/config"

// Load configuration (searches default paths)
cfg, err := config.Load("")
if err != nil {
    log.Fatal(err)
}

// Use configuration
fmt.Println("Network:", cfg.Network)
fmt.Println("Emulator gRPC Port:", cfg.Ports.Emulator.GRPC)
```

### Load from Specific Path

```go
cfg, err := config.Load("/path/to/aether.yaml")
```

### Environment Variable Overrides

```bash
# Override network mode
export AETHER_NETWORK=testnet

# Override emulator gRPC port
export AETHER_PORTS_EMULATOR_GRPC=4000

# Override global log level
export AETHER_LOGGING_LEVEL_GLOBAL=debug
```

## Configuration File Locations

The loader searches for `aether.yaml` in the following order:

1. Explicit path via `--config` flag (if provided to Load)
2. `./aether.yaml` (current directory)
3. `./config/aether.yaml`
4. `~/.aether/config.yaml` (user home)
5. `/etc/aether/config.yaml` (system-wide)

If no config file is found, sensible defaults are used.

## Configuration Structure

See `aether.example.yaml` in the project root for a complete example with all options documented.

### Main Sections

- **network**: Network mode (emulator, testnet, mainnet)
- **indexer**: Transaction indexing settings and data formatting
- **ports**: All port configurations in one place
- **evm**: EVM gateway settings
- **logging**: Log levels per component and output settings
- **ui**: Terminal UI preferences and defaults

## Validation

The configuration is validated on load:

- Network mode must be valid (emulator, testnet, mainnet)
- All ports must be in valid range (1-65535)
- Port conflicts are detected
- Log levels must be valid (trace, debug, info, warn, error, fatal)
- UI percentages must be 0-100
- History limits must be positive

## Defaults

All defaults match the current hardcoded behavior:

- Network: `emulator`
- Polling interval: `200ms`
- Emulator gRPC port: `3569`
- EVM RPC port: `8545`
- Global log level: `info`
- EVM gateway log level: `error`
- Max transactions: `10000`
- Max events: `10000`
- And more...

See `pkg/config/defaults.go` for complete defaults.

## Testing

Run tests with:

```bash
go test ./pkg/config/... -v
```

Tests cover:
- Default configuration
- Loading from file
- Environment variable overrides
- Validation (network, ports, log levels, UI settings)
- Log level inheritance
