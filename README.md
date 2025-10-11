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

## Testing EVM Support with Foundry

Aether automatically starts the Flow EVM Gateway, allowing you to deploy and interact with Solidity smart contracts using Foundry.

### Prerequisites

Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

### Quick Start: Deploy a Counter Contract

1. **Start Aether** in your Flow project directory:
   ```bash
   aether
   ```

2. **Create a new Foundry project** in a separate terminal:
   ```bash
   mkdir evm-test && cd evm-test
   forge init --no-git
   ```

3. **Create a simple Counter contract** at `src/Counter.sol`:
   ```solidity
   // SPDX-License-Identifier: MIT
   pragma solidity ^0.8.0;

   contract Counter {
       uint256 public count;

       function increment() public {
           count += 1;
       }

       function getCount() public view returns (uint256) {
           return count;
       }
   }
   ```

4. **Deploy the contract** to Flow EVM (running on localhost:3000):
   ```bash
   forge create --chain-id 646 --rpc-url http://localhost:3000 \
     --private-key 0x0000000000000000000000000000000000000000000000000000000000000001 \
     src/Counter.sol:Counter
   ```

5. **Interact with the deployed contract**:
   ```bash
   # Get the current count (should be 0)
   cast call <CONTRACT_ADDRESS> "getCount()(uint256)" \
     --chain-id 646 \
     --rpc-url http://localhost:3000

   # Increment the counter
   cast send <CONTRACT_ADDRESS> "increment()" \
     --chain-id 646 \
     --rpc-url http://localhost:3000 \
     --private-key 0x0000000000000000000000000000000000000000000000000000000000000001

   # Get the new count (should be 1)
   cast call <CONTRACT_ADDRESS> "getCount()(uint256)" \
     --chain-id 646 \
     --rpc-url http://localhost:3000
   ```

### Available Services and Ports

When Aether is running, the following services are available:

- **Flow Emulator (gRPC)**: `localhost:3569` - Flow Access Node API
- **Flow Emulator (REST)**: `localhost:8888` - Flow REST API
- **Flow Emulator (Admin)**: `localhost:8080` - Admin API
- **Flow Emulator (Debugger)**: `localhost:2345` - Debug Adapter Protocol
- **Dev Wallet**: `localhost:8701` - FCL Dev Wallet
- **EVM Gateway (JSON-RPC)**: `localhost:3000` - Ethereum JSON-RPC API

### Notes

- The EVM Gateway database is automatically cleaned up when you stop Aether
- The default private key used above is for testing purposes only
- All EVM transactions will appear in the Aether logs and transaction view
- The EVM network ID is set to Flow EVM Preview Net (Chain ID: **646**)

## Planned features

- [ ] show accounts
- [ ] evm support
- [ ] support for running on testnet/mainnet from the latest blockheight?
