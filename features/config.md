# Aether Configuration Design

## Overview

This document outlines the design for moving Aether's hardcoded configuration values into a structured, file-based configuration system. The goal is to provide a maintainable, readable configuration approach that scales with the application's complexity.

## Configuration Format

### Primary Format: YAML
**Rationale:**
- Highly readable with clear hierarchical structure
- Supports comments for documentation
- Native support for complex nested structures
- Wide tooling support in Go ecosystem
- Human-friendly for manual editing

### Fallback Format: JSON
**Rationale:**
- Machine-readable and widely supported
- Easy to generate programmatically
- Strict validation
- Good for CI/CD and automated deployments

### Implementation Strategy
- Use `viper` library for configuration management
  - Supports multiple formats (YAML, JSON, TOML)
  - Environment variable overrides
  - Configuration watching/hot-reload capability
  - Sensible defaults with override capability

### Configuration File Discovery
Priority order:
1. Explicit path via `--config` flag
2. `./aether.yaml` (current directory)
3. `./config/aether.yaml`
4. `~/.aether/config.yaml` (user home)
5. `/etc/aether/config.yaml` (system-wide)

## Configuration Structure

### Top-Level Sections

```
aether:
  network:      # Flow network configuration
  emulator:     # Local emulator settings
  evm:          # EVM gateway configuration
  indexer:      # Transaction indexing settings
  ui:           # Terminal UI preferences
  logging:      # Logging configuration
  paths:        # File system paths
  performance:  # Performance tuning
  security:     # Security settings
```

## Current Hardcoded Values to Extract

### 1. Network Configuration (`network`)

**Currently hardcoded in:**
- `pkg/aether/server.go`
- `pkg/flow/gateway.go`

**Values to extract:**
- **Network selection**: Currently via command-line only ("testnet", "mainnet", or empty for emulator)
- **Access node host**: Hardcoded as `localhost:3569` for emulator
- **Access node hosts for previous sporks**: Currently nil
- **Flow network ID**: Hardcoded as `flow-emulator` / `flowGo.Emulator`
- **Initial flow balance for new users**: Hardcoded as `1000.0` FLOW
- **Starting block height**: Currently `1` for emulator, latest for networks
- **Polling interval**: Currently `200ms` for emulator/testnet, `800ms` for mainnet

### 2. Emulator Configuration (`emulator`)

**Currently hardcoded in:**
- `pkg/aether/server.go`
- `pkg/flow/emulator.go`

**Values to extract:**
- **Auto-start emulator**: Currently always uses existing emulator
- **Emulator port**: Implicitly `3569`
- **Emulator admin port**: Not configurable
- **Service account setup**: Automatic
- **Contract deployment**: Automatic via `InitializeContracts`
- **Account creation**: Automatic via `CreateAccountsE`
- **Transaction folder name**: Hardcoded as `"aether"`
- **Base path options**: `""` or `"cadence"`
- **FCL contract deployment**: Automatic
- **Init transaction execution**: Automatic from `aether/` or `cadence/aether/`

### 3. EVM Gateway Configuration (`evm`)

**Currently hardcoded in:**
- `pkg/flow/gateway.go`

**Values to extract:**
- **Database directory**: Hardcoded as `./evm-gateway-db`
- **RPC port**: Hardcoded as `8545`
- **RPC host**: Hardcoded as `""` (all interfaces)
- **WebSocket enabled**: Hardcoded as `true`
- **EVM network/chain ID**: Hardcoded as `types.FlowEVMPreviewNetChainID` (646)
- **Coinbase address**: Derived from private key
- **COA (Cadence Owned Account) address**: Uses service account
- **Gas price**: Hardcoded as `0` (free)
- **Enforce gas price**: Hardcoded as `true`
- **Wallet enabled**: Hardcoded as `true`
- **Transaction state validation**: Hardcoded as `"local-index"`
- **Profiler settings**:
  - Enabled: `true`
  - Host: `"localhost"`
  - Port: `6060`
- **Metrics port**: Hardcoded as `9091`
- **Filter expiry**: Hardcoded as `300000000000` (300 seconds in nanoseconds)
- **Rate limiting**: Hardcoded as `0` (disabled)
- **Transaction request limits**:
  - Limit: `0` (unlimited)
  - Duration: `300000000000` (300 seconds)
- **Transaction batching**:
  - Mode: `false`
  - Interval: `1200000000` (1.2 seconds)
- **EOA activity cache TTL**: Hardcoded as `10000000000` (10 seconds)
- **Index-only mode**: Hardcoded as `false`

### 4. Indexer Configuration (`indexer`)

**Currently hardcoded in:**
- `pkg/flow/indexer.go`

**Values to extract:**
- **Polling interval**: Currently set per network (see network section)
- **Starting height strategy**: `1` for emulator, `latest` for networks
- **Block processing timeout**: Not explicitly set
- **Retry behavior**: Implicit continue on error
- **Context cancellation handling**: Hardcoded string matching
- **Collection key not found handling**: Implicit continue
- **Transaction filtering**: None (processes all)
- **Event filtering**: None (captures all)

### 5. UI Configuration (`ui`)

**Currently hardcoded in:**
- `pkg/ui/transactions_view.go`
- `pkg/ui/events_view.go`
- `pkg/ui/model.go`

**Values to extract:**
- **Max transactions to keep**: Hardcoded as `10000`
- **Max events to keep**: Hardcoded as `10000`
- **Default view**: Currently transactions view
- **Color scheme**: Solarized (hardcoded in colors)
- **Table split ratios**:
  - Transactions: 40% table, 60% detail
  - Events: 60% table, 40% detail
- **Default display modes**:
  - Show event fields: `false`
  - Show raw addresses: `false` (friendly names by default)
  - Full detail mode: `false`
- **Filter settings**:
  - Filter char limit: `50`
  - Filter width: `50`
- **Save dialog settings**:
  - Default transaction directory: `"transactions"` or `"cadence/transactions"`
  - Filename char limit: `50`
  - Width: `40`

### 6. Logging Configuration (`logging`)

**Currently hardcoded in:**
- `main.go`
- `pkg/logs/logger.go`

**Values to extract:**
- **Log level**: Currently set via `--log-level` flag (default: `info`)
- **Log format**: Console with color
- **Timestamp format**: `"3:04PM"`
- **Output destination**: `os.Stderr`
- **Caller information**: Enabled
- **Pretty printing**: Enabled
- **Log file output**: Not currently supported
- **Structured logging fields**: Hardcoded per component

### 7. Paths Configuration (`paths`)

**Currently hardcoded in:**
- `pkg/aether/server.go`
- `pkg/flow/gateway.go`
- `main.go`

**Values to extract:**
- **Aether directory paths**: `"aether"` or `"cadence/aether"`
- **Base path**: `""` or `"cadence"`
- **EVM gateway database**: `"./evm-gateway-db"`
- **Transaction save directory**: `"transactions"` or `"cadence/transactions"`
- **FCL contract path**: Passed as byte array
- **Log output directory**: Not configurable (stderr only)

### 8. Performance Configuration (`performance`)

**Currently hardcoded in:**
- Various files

**Values to extract:**
- **Goroutine pool sizes**: Not explicitly limited
- **Channel buffer sizes**: `BlockResult` channel is unbuffered
- **Pre-rendering**: Enabled for transaction/event details
- **Async rendering**: Enabled via goroutines
- **Cache sizes**:
  - Account registry: In-memory, no limit
  - Pre-rendered details: Stored per transaction/event
- **Batch processing**: Not implemented
- **Concurrent transaction processing**: Sequential per block

### 9. Security Configuration (`security`)

**Currently hardcoded in:**
- `pkg/flow/gateway.go`

**Values to extract:**
- **Private key management**:
  - EVM private key: Generated or loaded from file
  - Flow private key: From service account
  - Key file paths: Not configurable
- **Address validation**: Implicit
- **Rate limiting**: Disabled (`0`)
- **CORS settings**: Not configured
- **TLS/SSL**: Not configured
- **API authentication**: Not implemented

### 10. Underflow Options (`underflow`)

**Currently hardcoded in:**
- `pkg/aether/server.go`

**Values to extract:**
- **Byte array as hex**: Hardcoded as `true`
- **Show Unix timestamps as string**: Hardcoded as `true`
- **Timestamp format**: Hardcoded as `"2006-01-02 15:04:05 UTC"`

## Configuration Nesting Strategy

### Logical Component Hierarchy

```yaml
network:  emulator | testnet | mainnet

flow:
    new_user_balance: float64  # Default: 1000.0
    block_time: duration       # Default: 1s

indexer:
    polling_interval: duration
    underflow:
        byte_array_as_hex: bool
        show_timestamps_as_date: bool
        timestamp_format: string

ports:
  emulator: 
    grpc: int          # Default: 3569
    rest: int          # Default: 8888
    admin: int         # Default: 8080
    debugger: int      # Default: 2345
  dev_wallet: int      # Default: 8701
  evm:
    rpc: int           # Default: 8545
    profiler: int      # Default: 6060
    metrics: int       # Default: 9091

evm:
    database_path: string #Default: evm-gateway-db
    delete_database_on_start: bool #Default: true

logging:
  level:
    global: trace | debug | info | warn | error | fatal  # Default: info
    aether: trace | debug | info | warn | error | fatal  # Default: inherits global
    emulator: trace | debug | info | warn | error | fatal  # Default: inherits global
    dev_wallet: trace | debug | info | warn | error | fatal  # Default: inherits global
    evm_gateway: trace | debug | info | warn | error | fatal  # Default: error
  timestamp_format: string  # Default: "15:04:05"
  color: bool  # Default: true
  file:
    enabled: bool  # Default: false
    path: string  # Default: ""
    buffer_size: int  # Default: 1000

ui:
  theme: solarized | default | custom
  history:
    max_transactions: int
    max_events: int
    max_log_lines: int
  layout:
    default_view: transactions | events | runner
    transactions:
      table_width_percent: int
      detail_width_percent: int
    events:
      table_width_percent: int
      detail_width_percent: int
  defaults:
    show_event_fields: bool
    show_raw_addresses: bool
    full_detail_mode: bool
  filter:
    char_limit: int
    width: int
  save:
    default_directory: string
    filename_char_limit: int
    dialog_width: int
```

## Implementation Phases

### Phase 1: Core Configuration
- Set up viper configuration system
- Define base configuration struct
- Implement network and emulator configuration
- Add configuration file discovery
- Support environment variable overrides

### Phase 2: Component Configuration
- Add EVM gateway configuration
- Add indexer configuration
- Add UI configuration
- Add logging configuration

### Phase 3: Advanced Features
- Add paths configuration
- Add performance tuning
- Add security configuration
- Configuration validation
- Schema documentation

### Phase 4: Developer Experience
- Generate example configuration files
- Add configuration migration tool
- Add configuration validation CLI command
- Documentation and examples

## Configuration Validation

### Required Validations
- Network mode must be valid (emulator, testnet, mainnet)
- Ports must be valid ranges (1-65535)
- Paths must be valid and accessible
- Durations must be positive
- Numeric limits must be reasonable
- Mutually exclusive options checked

### Default Values Strategy
- All configuration should have sensible defaults
- Defaults should match current hardcoded behavior
- Minimal configuration should "just work"
- Advanced users can override as needed

## Migration Strategy

### Backward Compatibility
- Keep existing command-line flags working
- Command-line flags override configuration file
- Environment variables override configuration file
- Flags > Env Vars > Config File > Defaults

### Documentation
- Provide complete example configuration files
- Document all configuration options
- Provide migration guide from flags to config
- Include common configuration recipes

## Benefits

1. **Scalability**: Easy to add new configuration options without flag proliferation
2. **Readability**: YAML provides clear, documented configuration
3. **Maintainability**: Centralized configuration management
4. **Flexibility**: Different configs for different environments
5. **Validation**: Strong typing and validation at startup
6. **Documentation**: Configuration file serves as documentation
7. **Version Control**: Configuration can be versioned and shared
8. **Testing**: Easy to create test configurations
9. **Deployment**: Simplified deployment with environment-specific configs
10. **Discoverability**: All options visible in example config file
