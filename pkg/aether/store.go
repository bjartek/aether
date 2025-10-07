package aether

import (
	"sync"

	"github.com/bjartek/aether/pkg/flow"
)

// Store holds block results in memory with thread-safe access
type Store struct {
	mu     sync.RWMutex
	blocks []flow.BlockResult
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	return &Store{
		blocks: make([]flow.BlockResult, 0),
	}
}

// Add stores a new block result
func (s *Store) Add(block flow.BlockResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks = append(s.blocks, block)
}

// GetAll returns all stored blocks
func (s *Store) GetAll() []flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make([]flow.BlockResult, len(s.blocks))
	copy(result, s.blocks)
	return result
}

// GetLatest returns the most recent N blocks
func (s *Store) GetLatest(n int) []flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.blocks) == 0 {
		return []flow.BlockResult{}
	}
	
	start := len(s.blocks) - n
	if start < 0 {
		start = 0
	}
	
	result := make([]flow.BlockResult, len(s.blocks)-start)
	copy(result, s.blocks[start:])
	return result
}

// GetByHeight returns a block by its height
func (s *Store) GetByHeight(height uint64) *flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for i := range s.blocks {
		if s.blocks[i].Block.Height == height {
			return &s.blocks[i]
		}
	}
	return nil
}

// Count returns the total number of stored blocks
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}

// Clear removes all stored blocks
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks = make([]flow.BlockResult, 0)
}
