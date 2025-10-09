package flow

import (
	"encoding/json"
	"os"
)

// TransactionConfig represents a JSON configuration for running a transaction or script
type TransactionConfig struct {
	Name      string            `json:"name"`      // Name of the transaction/script file (without .cdc extension)
	Signers   []string          `json:"signers"`   // List of signer names (friendly names from flow.json)
	Arguments map[string]string `json:"arguments"` // Map of argument name to value
}

// LoadTransactionConfig loads a transaction configuration from a JSON file
func LoadTransactionConfig(path string) (*TransactionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config TransactionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveTransactionConfig saves a transaction configuration to a JSON file
func SaveTransactionConfig(path string, config *TransactionConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
