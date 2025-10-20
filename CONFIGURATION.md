# Aether Configuration Guide

## Overview

Aether now uses a flexible configuration system that allows you to customize behavior through configuration files instead of command-line flags.

## Quick Start

### Using Default Configuration

Simply run Aether without any configuration file:

```bash
./aether
```

This will use sensible defaults for all settings.

### Using a Configuration File

Create an `aether.yaml` file in your project directory:

```bash
cp aether.example.yaml aether.yaml
# Edit aether.yaml with your preferences
./aether
```

Or specify a custom config path:

```bash
./aether --config /path/to/my-config.yaml
```

## Configuration File Locations

Aether searches for `aether.yaml` in the following order:

1. Path specified via `--config` flag
2. `./aether.yaml` (current directory)
3. `./config/aether.yaml`
4. `~/.aether/config.yaml` (user home)
5. `/etc/aether/config.yaml` (system-wide)

## Command-Line Overrides

Command-line flags override configuration file settings:

```bash
# Override network mode
./aether -n testnet

# Or specify a custom config
./aether --config my-config.yaml
```

## Environment Variables

Any configuration value can be overridden with environment variables using the `AETHER_` prefix:

```bash
# Override network
export AETHER_NETWORK=testnet

# Override log level
export AETHER_LOGGING_LEVEL_GLOBAL=debug

# Override emulator gRPC port
export AETHER_PORTS_EMULATOR_GRPC=4000
```

## Configuration Sections

### Network

Controls which Flow network to connect to:

```yaml
network: emulator  # emulator, testnet, or mainnet
```

### Flow

Flow blockchain settings:

```yaml
flow:
  new_user_balance: 1000.0  # Initial FLOW tokens for new accounts
  block_time: 1s            # Block production interval (emulator only)
```

### Indexer

Transaction indexing and data formatting:

```yaml
indexer:
  polling_interval: 200ms
  underflow:
    byte_array_as_hex: true
    show_timestamps_as_date: true
    timestamp_format: "2006-01-02 15:04:05 UTC"
```

### Ports

All port configurations in one place to avoid conflicts:

```yaml
ports:
  emulator:
    grpc: 3569
    rest: 8888
    admin: 8080
    debugger: 2345
  dev_wallet: 8701
  evm:
    rpc: 8545
    profiler: 6060
    metrics: 9091
```

### EVM Gateway

EVM gateway specific settings:

```yaml
evm:
  database_path: evm-gateway-db
  delete_database_on_start: true
```

### Logging

Per-component log levels and output settings:

```yaml
logging:
  level:
    global: info          # Default for all components
    aether: info          # Inherits global if not set
    emulator: info        # Inherits global if not set
    dev_wallet: info      # Inherits global if not set
    evm_gateway: error    # Reduced noise from EVM gateway
  timestamp_format: "15:04:05"
  color: true
  file:
    enabled: false
    path: ""
    buffer_size: 1000
```

### UI

Terminal UI preferences:

```yaml
ui:
  theme: solarized
  history:
    max_transactions: 10000
    max_events: 10000
    max_log_lines: 10000
  layout:
    default_view: transactions  # transactions, events, or runner
    transactions:
      table_width_percent: 40
      detail_width_percent: 60
    events:
      table_width_percent: 60
      detail_width_percent: 40
  defaults:
    show_event_fields: false
    show_raw_addresses: false
    full_detail_mode: false
  filter:
    char_limit: 50
    width: 50
  save:
    default_directory: transactions
    filename_char_limit: 50
    dialog_width: 40
```

## Examples

### Development Configuration

```yaml
# dev-config.yaml
network: emulator
logging:
  level:
    global: debug
    evm_gateway: warn
  file:
    enabled: true
    path: aether-dev.log
ui:
  defaults:
    show_event_fields: true
    full_detail_mode: true
```

### Testnet Configuration

```yaml
# testnet-config.yaml
network: testnet
indexer:
  polling_interval: 200ms
logging:
  level:
    global: info
ui:
  history:
    max_transactions: 50000
    max_events: 50000
```

### Mainnet Configuration

```yaml
# mainnet-config.yaml
network: mainnet
indexer:
  polling_interval: 800ms  # Slower polling for mainnet
logging:
  level:
    global: warn  # Less verbose for production
    evm_gateway: error
```

## Migration from Flags

Old command-line usage:

```bash
./aether -n testnet
```

New configuration file approach:

```yaml
# aether.yaml
network: testnet
```

Then simply run:

```bash
./aether
```

For logging configuration, use the config file:

```yaml
# aether.yaml
logging:
  level:
    global: debug
  file:
    enabled: true
    path: debug.log
```

## Validation

Configuration is validated on startup. Common errors:

- **Invalid network**: Must be `emulator`, `testnet`, or `mainnet`
- **Port conflicts**: Each port must be unique
- **Invalid log level**: Must be `trace`, `debug`, `info`, `warn`, `error`, or `fatal`
- **Invalid percentages**: Must be between 0 and 100

## See Also

- `aether.example.yaml` - Complete example with all options
- `pkg/config/README.md` - Developer documentation
- `features/config.md` - Design documentation
