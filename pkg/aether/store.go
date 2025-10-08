package aether

import (
	"sync"

	"github.com/bjartek/aether/pkg/flow"
)

// Store holds block results in memory with thread-safe access
// Only stores blocks that have transactions, indexed by height for fast lookup
type Store struct {
	mu      sync.RWMutex
	blocks  map[uint64]flow.BlockResult
	heights []uint64 // Sorted list of heights for ordered iteration
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	return &Store{
		blocks:  make(map[uint64]flow.BlockResult),
		heights: make([]uint64, 0),
	}
}

// Add stores a new block result if it has transactions
func (s *Store) Add(block flow.BlockResult) {
	// Skip blocks without transactions
	if len(block.Transactions) == 0 {
		block.Logger.Debug().Uint64("height", block.Block.Height).Msg("Skipping block with no transactions")
		return
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	height := block.Block.Height
	// Check if already stored
	if _, exists := s.blocks[height]; exists {
		return
	}
	
	s.blocks[height] = block
	s.heights = append(s.heights, height)
	block.Logger.Info().Int("totalBlocks", len(s.blocks)).Uint64("height", height).Msg("Block added to store")
}

// GetAll returns all stored blocks ordered by height
func (s *Store) GetAll() []flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]flow.BlockResult, 0, len(s.blocks))
	for _, height := range s.heights {
		if block, ok := s.blocks[height]; ok {
			result = append(result, block)
		}
	}
	return result
}

// GetLatest returns the most recent N blocks
func (s *Store) GetLatest(n int) []flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.heights) == 0 {
		return []flow.BlockResult{}
	}
	
	start := len(s.heights) - n
	if start < 0 {
		start = 0
	}
	
	result := make([]flow.BlockResult, 0, len(s.heights)-start)
	for i := start; i < len(s.heights); i++ {
		if block, ok := s.blocks[s.heights[i]]; ok {
			result = append(result, block)
		}
	}
	return result
}

// GetByHeight returns a block by its height
func (s *Store) GetByHeight(height uint64) *flow.BlockResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if block, ok := s.blocks[height]; ok {
		result := block
		return &result
	}
	return nil
}

// Count returns the total number of stored blocks
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.heights)
}

// Clear removes all stored blocks
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks = make(map[uint64]flow.BlockResult)
	s.heights = make([]uint64, 0)
}
