# Makefile for Flow EVM Counter Contract
# This assumes Aether is running on localhost:3000

# Configuration
RPC_URL := http://localhost:3000
PRIVATE_KEY := 0x0000000000000000000000000000000000000000000000000000000000000001
CONTRACT := src/Counter.sol:Counter

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: help build test deploy clean call-count increment get-balance install

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

install: ## Install Foundry dependencies
	@echo "$(YELLOW)Installing dependencies...$(NC)"
	forge install

build: ## Build the contracts
	@echo "$(YELLOW)Building contracts...$(NC)"
	forge build

test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	forge test -vv

deploy: build ## Deploy Counter contract
	@echo "$(YELLOW)Deploying Counter contract to Flow EVM (chain ID 646)...$(NC)"
	@OUTPUT=$$(forge create --rpc-url $(RPC_URL) --chain-id 646 --private-key $(PRIVATE_KEY) $(CONTRACT) 2>&1); \
	echo "$$OUTPUT"; \
	ADDRESS=$$(echo "$$OUTPUT" | grep "Deployed to:" | awk '{print $$3}'); \
	if [ -n "$$ADDRESS" ]; then \
		echo "$$ADDRESS" > .contract-address; \
		echo "$(GREEN)Contract deployed at: $$ADDRESS$(NC)"; \
		echo "$(GREEN)Address saved to .contract-address$(NC)"; \
	fi

call-count: ## Get the current count value
	@if [ ! -f .contract-address ]; then \
		echo "$(YELLOW)No contract address found. Please deploy first with 'make deploy'$(NC)"; \
		exit 1; \
	fi
	@CONTRACT_ADDR=$$(cat .contract-address); \
	echo "$(YELLOW)Calling getCount() on $$CONTRACT_ADDR...$(NC)"; \
	cast call $$CONTRACT_ADDR "getCount()(uint256)" --rpc-url $(RPC_URL) --chain-id 646

increment: ## Increment the counter
	@if [ ! -f .contract-address ]; then \
		echo "$(YELLOW)No contract address found. Please deploy first with 'make deploy'$(NC)"; \
		exit 1; \
	fi
	@CONTRACT_ADDR=$$(cat .contract-address); \
	echo "$(YELLOW)Incrementing counter on $$CONTRACT_ADDR...$(NC)"; \
	cast send $$CONTRACT_ADDR "increment()" --rpc-url $(RPC_URL) --chain-id 646 --private-key $(PRIVATE_KEY)

demo: deploy ## Deploy and run a quick demo (deploy, check count, increment, check again)
	@echo "$(GREEN)=== Counter Contract Demo ===$(NC)"
	@sleep 2
	@echo "\n$(YELLOW)Initial count:$(NC)"
	@make call-count
	@sleep 1
	@echo "\n$(YELLOW)Incrementing...$(NC)"
	@make increment
	@sleep 2
	@echo "\n$(YELLOW)New count:$(NC)"
	@make call-count
	@echo "\n$(GREEN)Demo complete!$(NC)"

get-balance: ## Get the balance of the deployer account
	@echo "$(YELLOW)Getting balance...$(NC)"
	@cast balance 0x0000000000000000000000000000000000000001 --rpc-url $(RPC_URL)

chain-id: ## Get the chain ID
	@echo "$(YELLOW)Getting chain ID...$(NC)"
	@cast chain-id --rpc-url $(RPC_URL)

block-number: ## Get the current block number
	@echo "$(YELLOW)Getting block number...$(NC)"
	@cast block-number --rpc-url $(RPC_URL)

clean: ## Clean build artifacts and remove contract address
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	forge clean
	rm -f .contract-address
	@echo "$(GREEN)Clean complete!$(NC)"

fmt: ## Format Solidity code
	@echo "$(YELLOW)Formatting code...$(NC)"
	forge fmt
