# ÆTHER - Elevate you flow cli dev experience

Aether was made to make it possible to do cadence based development using the emulator and associated tools in an easy way.

Instead of having to start up 3-4 different tools and configure them and then run lots of transactions to set up your workspace you can just configure aether an run `aether`

see casts

- iteration 1: <https://asciinema.org/a/rXpJIbBdELLwfQAy7Y6JZidiS>
- iteration 2: <http://asciinema.org/a/zwDZZTHTwiu50MucxAc3vVR3h> help
- iteration 3: <https://asciinema.org/a/0BeMtZblr0kEbQ8iLz0ZQoIMY> events
- iteration 4: <https://asciinema.org/a/U6cj7zVq9998NBOeu3ZlD3Ryz> execute
- iteration 5: <https://asciinema.org/a/4XBuCXmxXcenpYpmGR7uSFApd> save interactions

major rework

- iteration 6: <https://asciinema.org/a/wm8IE058jBsszh372zM5fRdul> full demo

## Install

It is currently not possible to install aether with go install because we use my fork of flow-evm-gateway since some issues are not solved yet.

- check out this project
- run `go install`
- navigate to a folder with flow.json
- run `aether`
  For troubleshooting and reporting issues, see [DEBUGGING.md](DEBUGGING.md).

## Configuration

Aether can be configured using viper with an `aether.yaml` file . If no configuration file is present, sensible defaults are used (see [pkg/config/defaults.go](pkg/config/defaults.go)).

see [aether.full.yaml](aether.full.yaml) for a full example

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
- runs a set of init transactions from aether or cadence/aether folder, or you can specify your own or chose at startup
  - transactions are run in alphabetical order
  - signer is taken from names in flow.json without emulator- prefix
    - so `(alice: &Account)` means sign with alice
  - can also run saved/templated transctions with given sender and arguments (json file)
- optionally start your frontend and weave in the logs

### Mainnet/testnet use

- follows mainnet or testnet
- allows to save a transaction to run be run later
- allows the user to run transaction if key configured in flow.json
