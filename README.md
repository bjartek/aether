# ÆTHER

Elevate you flow cli dev experience

see casts

- iteration 1: https://asciinema.org/a/rXpJIbBdELLwfQAy7Y6JZidiS
- iteration 2: http://asciinema.org/a/zwDZZTHTwiu50MucxAc3vVR3h  help
- iteration 3: https://asciinema.org/a/0BeMtZblr0kEbQ8iLz0ZQoIMY events
- iteration 4:     https://asciinema.org/a/U6cj7zVq9998NBOeu3ZlD3Ryz execute
- iteration 5:     https://asciinema.org/a/4XBuCXmxXcenpYpmGR7uSFApd save interactions
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
  - shows events in a table
  - shows event details in an inspector
  - shows log from emulator and dev-wallet and aether
  - shows a dashboard
  - filter transactions
  - filter events
  - filter logs
  - save interactions
  - run interactions
  - show events

## Planned features

- [x] save transactions from transactions view with network suffix
- [ ] show accounts
- [ ] evm support
- [ ] possibly fix keybindings for running tx, they are a bit clunky
- [ ] support for running on testnet/mainnet from the latest blockheight?
