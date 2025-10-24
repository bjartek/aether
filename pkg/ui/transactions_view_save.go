package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bjartek/aether/pkg/flow"
)

// saveTransaction saves a transaction to .cdc and .json files
func (tv *TransactionsView) saveTransaction(filename string) string {
	// Get current transaction
	currentIdx := tv.sv.GetCursor()
	if currentIdx < 0 || currentIdx >= len(tv.transactions) {
		return "No transaction selected"
	}
	
	tx := tv.transactions[currentIdx]
	
	// Get network from overflow state (default to emulator if not available)
	network := "emulator"
	if tv.overflow != nil {
		network = tv.overflow.Network.Name
	}
	
	// Always use transactions directory, create if needed
	dir := "transactions"

	// Check if cadence/transactions exists instead
	if _, err := os.Stat("cadence/transactions"); err == nil {
		dir = "cadence/transactions"
	} else if _, err := os.Stat("transactions"); os.IsNotExist(err) {
		// Neither exists, create transactions
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Sprintf("failed to create directory %s: %v", dir, err)
		}
	}

	// Save .cdc file with network suffix (e.g., .emulator.cdc, .testnet.cdc)
	cdcFilename := filename + "." + network + ".cdc"
	cdcPath := filepath.Join(dir, cdcFilename)
	if err := os.WriteFile(cdcPath, []byte(tx.Script), 0644); err != nil {
		return fmt.Sprintf("failed to write %s: %v", cdcPath, err)
	}

	// Build JSON config with arguments (but empty signers)
	config := &flow.TransactionConfig{
		Name:      filename + "." + network,
		Signers:   []string{}, // Leave empty as requested
		Arguments: make(map[string]interface{}),
	}

	// Populate arguments from transaction data
	for _, arg := range tx.Arguments {
		// Convert to string for JSON config
		config.Arguments[arg.Name] = fmt.Sprintf("%v", arg.Value)
	}

	// Save JSON config file
	jsonFilename := filename + ".json"
	jsonPath := filepath.Join(dir, jsonFilename)
	if err := flow.SaveTransactionConfig(jsonPath, config); err != nil {
		return fmt.Sprintf("failed to write %s: %v", jsonPath, err)
	}

	return "" // Success
}
