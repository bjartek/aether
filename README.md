# ÆTHER

Elevate you flow cli dev experience

see casts

- iteration 1: <https://asciinema.org/a/rXpJIbBdELLwfQAy7Y6JZidiS>
- iteration 2: <http://asciinema.org/a/zwDZZTHTwiu50MucxAc3vVR3h> help
- iteration 3: <https://asciinema.org/a/0BeMtZblr0kEbQ8iLz0ZQoIMY> events
- iteration 4: <https://asciinema.org/a/U6cj7zVq9998NBOeu3ZlD3Ryz> execute
- iteration 5: <https://asciinema.org/a/4XBuCXmxXcenpYpmGR7uSFApd> save interactions

## How to

- go build the binary, or install it with go install
- navigate to a folder with flow.json
- run `aether`

### Command Line Options

```bash
# Run with default settings
aether

# Specify a custom configuration file
aether --config path/to/aether.yaml

# Override network setting (emulator, testnet, or mainnet)
aether -n testnet

# Enable debug logging (creates aether-debug.log)
aether --debug
# or
aether -d
```

For troubleshooting and reporting issues, see [DEBUGGING.md](DEBUGGING.md).

## Configuration

Aether can be configured using an `aether.yaml` file in your project directory. If no configuration file is present, sensible defaults are used.

### Configuration File Location

Aether looks for `aether.yaml` in the current directory where you run the command.

### Configuration Structure

```yaml
# Network to use (default: "emulator")
network: emulator

# Flow blockchain settings
flow:
  new_user_balance: 1000.0      # Initial balance for new accounts (FLOW tokens)
  block_time: 1s                # Block production time

# Indexer settings for monitoring blockchain events
indexer:
  polling_interval: 200ms       # How often to poll for new blocks
  underflow:
    byte_array_as_hex: true     # Display byte arrays as hex strings
    show_timestamps_as_date: true  # Show timestamps in human-readable format
    timestamp_format: "2006-01-02 15:04:05 UTC"  # Go time format string

# Port configurations
ports:
  emulator:
    grpc: 3569                  # Flow emulator gRPC port
    rest: 8888                  # Flow emulator REST API port
    admin: 8080                 # Flow emulator admin port
    debugger: 2345              # Flow emulator debugger port
  dev_wallet: 8701              # Dev wallet port
  evm:
    rpc: 8545                   # EVM gateway JSON-RPC port
    profiler: 6060              # EVM gateway profiler port
    metrics: 9091               # EVM gateway metrics port

# EVM gateway settings
evm:
  database_path: "evm-gateway-db"  # Path to EVM gateway database
  delete_database_on_start: true   # Clear database on startup

# Logging configuration
logging:
  level:
    global: info                # Global log level (debug, info, warn, error)
    aether: ""                  # Aether log level (empty = inherit global)
    emulator: ""                # Emulator log level (empty = inherit global)
    dev_wallet: ""              # Dev wallet log level (empty = inherit global)
    evm_gateway: error          # EVM gateway log level
  timestamp_format: "15:04:05"  # Time format for log timestamps
  color: true                   # Enable colored output
  file:
    enabled: false              # Enable file logging
    path: ""                    # Log file path (e.g., "aether.log")
    buffer_size: 1000           # Log buffer size

# UI preferences
ui:
  history:
    max_transactions: 10000     # Maximum transactions to keep in history
    max_events: 10000           # Maximum events to keep in history
    max_log_lines: 10000        # Maximum log lines to keep in history
  
  layout:
    transactions_split_percent: 40  # Table width as percentage (0-100)
    events_split_percent: 50        # Table width as percentage (0-100)
    runner_split_percent: 40        # Table width as percentage (0-100)
  
  defaults:
    show_event_fields: true     # Show event fields in transaction details
    show_raw_addresses: false   # Show raw addresses instead of account names
    time_format: "15:04:05"     # Time format for UI timestamps
```

### Minimal Configuration Example

Most users only need to override a few settings. Here's a minimal example:

```yaml
# aether.yaml - minimal configuration
logging:
  level:
    global: debug              # Enable debug logging
  file:
    enabled: true
    path: "aether.log"         # Log to file

ui:
  defaults:
    show_raw_addresses: true   # Show raw addresses
```

### Configuration Priority

1. Command-line flags (highest priority)
2. `aether.yaml` configuration file
3. Default values (lowest priority)

### Command-Line Overrides

- `--config <path>` - Specify custom configuration file location
- `-n <network>` - Override network setting (emulator, testnet, mainnet)
- `--debug` / `-d` - Enable debug logging (see [DEBUGGING.md](DEBUGGING.md))

## Local development

run `make` to build the binary start it and run it in the example folder

## Features

 - navgigate tabs with `<number>` or tabs/arrow keys.
 - show help in fotter with `?`
 - shows transactions in a tabular view with an inspecor, can see details with `enter` or `space`
  - can toggle to show human readable addresses with `a` 
  - can collapse/expand events with `e`
  - can show `[uint8]` arrays as hex configured in config file
  - can show unix_timestamps as human readable date, confiured in config file
  - can save an existing transaction with predefined arguments/signer. note that this is only valid for the current network

 - show events in a tabular view with an inspector, can see details
  - can toggle to show human readable addresses with `a` 
  - can show `[uint8]` arrays as hex configured in config file
  - can show unix_timestamps as human readable date, confiured in config file
 - show logs of all the components with log level configured in config file
 - shows a dashboard of what is exposed and what is run 
 - allows the user to run transactions

### Emulator use
- starts flow emulator on default port 3569
- starts dev-wallet at default port 8701
- starts EVM gateway on default port 3000 (JSON-RPC API)
- deploys all contracts in flow.json for emulator
- creates all users in flow.json that are mentioned in deploy block 
- mints flow tokens for all users specified amount in 
- runs a set of init transactions from aether or cadence/aether folder
  - transactions are run in alphabetical order
  - signer is taken from names in flow.json without emulator- prefix
    - so `(alice: &Account)` means sign with alice
  - can also run saved/templated transctions with given sender and arguments (json file)

### Mainnet/testnet use
 - follows mainnet or testnet
 - allows to save a transaction to run be run later
 - allows the user to run transaction if key configured in flow.json


