## Flow EVM Example with Foundry

This example demonstrates deploying and interacting with Solidity smart contracts on the Flow EVM using Foundry.

**Prerequisites:** Make sure Aether is running in the parent directory to have access to the Flow EVM Gateway on `localhost:3000` (Chain ID: 646).

## Quick Start with Make

```bash
# Show all available commands
make help

# Deploy the Counter contract
make deploy

# Get the current count
make call-count

# Increment the counter
make increment

# Run a complete demo (deploy + increment + check)
make demo
```

## Available Make Targets

- `make install` - Install Foundry dependencies
- `make build` - Build the contracts
- `make test` - Run tests
- `make deploy` - Deploy Counter contract (saves address to .contract-address)
- `make call-count` - Get the current count value
- `make increment` - Increment the counter
- `make demo` - Run a complete demo
- `make get-balance` - Get deployer account balance
- `make chain-id` - Get the chain ID (should show 646)
- `make block-number` - Get current block number
- `make clean` - Clean build artifacts
- `make fmt` - Format Solidity code

---

## Foundry

**Foundry is a blazing fast, portable and modular toolkit for Ethereum application development written in Rust.**

Foundry consists of:

-   **Forge**: Ethereum testing framework (like Truffle, Hardhat and DappTools).
-   **Cast**: Swiss army knife for interacting with EVM smart contracts, sending transactions and getting chain data.
-   **Anvil**: Local Ethereum node, akin to Ganache, Hardhat Network.
-   **Chisel**: Fast, utilitarian, and verbose solidity REPL.

## Documentation

https://book.getfoundry.sh/

## Usage

### Build

```shell
$ forge build
```

### Test

```shell
$ forge test
```

### Format

```shell
$ forge fmt
```

### Gas Snapshots

```shell
$ forge snapshot
```

### Anvil

```shell
$ anvil
```

### Deploy

```shell
$ forge script script/Counter.s.sol:CounterScript --rpc-url <your_rpc_url> --private-key <your_private_key>
```

### Cast

```shell
$ cast <subcommand>
```

### Help

```shell
$ forge --help
$ anvil --help
$ cast --help
```
