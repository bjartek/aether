# ÆTHER

Elevate you flow cli dev experience

see cast <https://asciinema.org/a/rXpJIbBdELLwfQAy7Y6JZidiS> for a quick demo

## How to

- go build the binary, or install it with go install
- navigate to a folder with flow.json
- run `aether`

### Command Line Flags

- `--verbose` - Enable verbose (debug) logging
- `--log-file <filename>` - Log to file (e.g. `aether --log-file aether-debug.log`)

**Examples:**
```bash
# Run with default settings (info level, no file logging)
aether

# Run with verbose logging
aether --verbose

# Run with file logging
aether --log-file debug.log

# Run with both verbose and file logging
aether --verbose --log-file debug.log
```

## Local development

run `make` to build the binary start it and run it in the example folder

## Features

- starts flow emulator on default port 3569
- starts dev-wallet at default port 8701
- deploys all contracts in flow.json for emulator
- creates all users in flow.json that are mentioned in deploy block
- runs a set of init transactions from aether or cadence/aether folder
  - transactions are run in alphabetical order
  - signer is taken from names in flow.json without emulator- prefix
    - so `(alice: &Account)` means sign with alice
  - no arguments are allowed in init transactions
- starts the aether TUI
  - shows transactions in a table
  - shows transaction details in an inspector
  - shows log from emulator and dev-wallet and aether
  - shows a dashboard
  - filter transactions
  - filter logs

## Planned features

- [ ] run transactions/script
- [ ] show events 
- [ ] filter events
- [ ] show accounts
- [x] show better help system
- [ ] evm support