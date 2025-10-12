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
- starts EVM gateway on default port 3000 (JSON-RPC API)
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
  - save transactions from transactions view with network suffix
  - syntax highlight using chroma
  -  support for running on testnet/mainnet from the latest blockheight?

## Testing EVM Support with Foundry

Aether automatically starts the Flow EVM Gateway, allowing you to deploy and interact with Solidity smart contracts using Foundry.

### Prerequisites

Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

### EVM notes
 This does not work atm, debuging with mpeter 
 
1. **Deploy the contract** to Flow EVM (running on localhost:3000):
   ```bash
   forge script script/Counter.s.sol:CounterScript \
        --rpc-url http://localhost:8545 \
        --slow \
        -vvv \
        --legacy \
        --broadcast \
        --via-ir \
        -i 1
   ```
   note down the address that is created, i think this is always 0xC7f2Cf4845C6db0e1a1e91ED41Bcd0FcC1b0E141

2. **Interact with the deployed contract**:
   ```bash
   # Get the current count (should be 0)
   cast call 0xC7f2Cf4845C6db0e1a1e91ED41Bcd0FcC1b0E141 "getCount()(uint256)" \
     --chain-id 646 \
     --rpc-url http://localhost:8545

   #this fails for me saying that there is no contract here

   # Increment the counter
   cast send  0xC7f2Cf4845C6db0e1a1e91ED41Bcd0FcC1b0E141 "increment()" \
     --chain-id 646 \
     --rpc-url http://localhost:8545 \
     --private-key 0x2619878f0e2ff438d17835c2a4561cb87b4d24d72d12ec34569acd0dd4af7c21

   # Get the new count (should be 1)
   cast call 0xC7f2Cf4845C6db0e1a1e91ED41Bcd0FcC1b0E141 "getCount()(uint256)" \
     --chain-id 646 \
     --rpc-url http://localhost:8545
   ```

## Planned features

- [ ] show accounts
- [ ] evm support
