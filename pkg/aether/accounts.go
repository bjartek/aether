package aether

import (
	"strings"
	"sync"

	"github.com/bjartek/overflow/v2"
	"github.com/onflow/flowkit/v2/config"
)

// AccountRegistry maintains a mapping of addresses to human-friendly names
type AccountRegistry struct {
	mu           sync.RWMutex
addressToName map[string]string
}

// NewAccountRegistry creates a new account registry from overflow state
func NewAccountRegistry(o *overflow.OverflowState) *AccountRegistry {
	registry := &AccountRegistry{
		addressToName: make(map[string]string),
	}
	
	if o == nil || o.State == nil {
		return registry
	}
	
	// Get all accounts for the emulator network using flowkit
	networkAccounts := o.State.AccountsForNetwork(config.EmulatorNetwork)
	if networkAccounts == nil {
		return registry
	}
	
	for _, account := range *networkAccounts {
		// Strip "emulator-" prefix if present
		friendlyName := strings.TrimPrefix(account.Name, "emulator-")
		
		// Override "account" to "service-account" for clarity
		if friendlyName == "account" {
			friendlyName = "service-account"
		}
		
		// All Flow addresses should have 0x prefix and lowercase
		// (Flow addresses are case-insensitive but we normalize to lowercase)
		normalizedAddr := strings.ToLower(normalizeAddress(account.Address.String()))
		
		// Store with 0x prefix and lowercase
		registry.addressToName[normalizedAddr] = friendlyName
	}
	
	return registry
}

// DebugDump logs all registered accounts for debugging
func (r *AccountRegistry) DebugDump() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy of the map for logging
	dump := make(map[string]string, len(r.addressToName))
	for addr, name := range r.addressToName {
		dump[addr] = name
	}
	return dump
}

// normalizeAddress ensures address has 0x prefix
func normalizeAddress(address string) string {
	if address == "" || address == "N/A" {
		return address
	}
	if !strings.HasPrefix(address, "0x") {
		return "0x" + address
	}
	return address
}

// GetName returns the friendly name for an address, or the address itself if not found
func (r *AccountRegistry) GetName(address string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if address == "" || address == "N/A" {
		return address
	}
	
	// Normalize to 0x prefix and lowercase for lookup
	normalizedAddr := strings.ToLower(normalizeAddress(address))
	if name, ok := r.addressToName[normalizedAddr]; ok {
		return name
	}
	
	return address
}

// FormatAddress returns "name (address)" if name is known, otherwise just address
func (r *AccountRegistry) FormatAddress(address string) string {
	if address == "" || address == "N/A" {
		return address
	}
	
	// Ensure address has 0x prefix for display
	displayAddr := normalizeAddress(address)
	name := r.GetName(displayAddr)
	
	// Debug logging
	// TODO: Remove once working
	if name != displayAddr {
		// Found a friendly name
		return name + " (" + displayAddr + ")"
	}
	
	// No friendly name found, return address only
	return displayAddr
}

// FormatAddressShort returns "name (addr...)" with truncated address if name is known
func (r *AccountRegistry) FormatAddressShort(address string, startLen, endLen int) string {
	if address == "" || address == "N/A" {
		return address
	}
	
	// Ensure address has 0x prefix
	displayAddr := normalizeAddress(address)
	name := r.GetName(address)
	
	if name != address && name != displayAddr {
		truncated := displayAddr
		if len(displayAddr) > startLen+endLen {
			truncated = displayAddr[:startLen] + "..." + displayAddr[len(displayAddr)-endLen:]
		}
		return name + " (" + truncated + ")"
	}
	return displayAddr
}

// GetAllNames returns a sorted list of all registered account names
func (r *AccountRegistry) GetAllNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.addressToName))
	for _, name := range r.addressToName {
		names = append(names, name)
	}
	
	return names
}
