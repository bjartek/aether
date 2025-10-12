#!/bin/bash

export ETH_FROM=0x3ca7971b5be71bcd2db6cedf667f0ae5e5022fed
forge script script/Counter.s.sol:CounterScript \
  --rpc-url http://localhost:8545 \
  --slow \
  -vvv \
  --legacy \
  --broadcast \
  --via-ir \
  -i 1
