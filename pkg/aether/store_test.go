package aether

import (
	"testing"
	"time"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	flowsdk "github.com/onflow/flow-go-sdk"
	"github.com/rs/zerolog"
)

func TestStore_OnlyStoresBlocksWithTransactions(t *testing.T) {
	store := NewStore()
	logger := zerolog.Nop()

	// Block without transactions - should not be stored
	emptyBlock := flow.BlockResult{
		Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: 1}},
		Transactions: []overflow.OverflowTransaction{},
		Logger:       logger,
		StartTime:    time.Now(),
	}
	store.Add(emptyBlock)

	if store.Count() != 0 {
		t.Errorf("Expected 0 blocks, got %d", store.Count())
	}

	// Block with transactions - should be stored
	blockWithTx := flow.BlockResult{
		Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: 2}},
		Transactions: []overflow.OverflowTransaction{{}}, // At least one transaction
		Logger:       logger,
		StartTime:    time.Now(),
	}
	store.Add(blockWithTx)

	if store.Count() != 1 {
		t.Errorf("Expected 1 block, got %d", store.Count())
	}
}

func TestStore_GetByHeight(t *testing.T) {
	store := NewStore()
	logger := zerolog.Nop()

	// Add blocks
	for i := uint64(1); i <= 5; i++ {
		store.Add(flow.BlockResult{
			Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: i}},
			Transactions: []overflow.OverflowTransaction{{}},
			Logger:       logger,
			StartTime:    time.Now(),
		})
	}

	// Test GetByHeight
	block := store.GetByHeight(3)
	if block == nil {
		t.Fatal("Expected to find block at height 3")
	}
	if block.Block.Height != 3 {
		t.Errorf("Expected height 3, got %d", block.Block.Height)
	}

	// Test non-existent height
	block = store.GetByHeight(999)
	if block != nil {
		t.Error("Expected nil for non-existent block")
	}
}

func TestStore_GetLatest(t *testing.T) {
	store := NewStore()
	logger := zerolog.Nop()

	// Add 10 blocks
	for i := uint64(1); i <= 10; i++ {
		store.Add(flow.BlockResult{
			Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: i}},
			Transactions: []overflow.OverflowTransaction{{}},
			Logger:       logger,
			StartTime:    time.Now(),
		})
	}

	// Get latest 3
	latest := store.GetLatest(3)
	if len(latest) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(latest))
	}

	// Verify they are the most recent
	if latest[0].Block.Height != 8 || latest[1].Block.Height != 9 || latest[2].Block.Height != 10 {
		t.Errorf("Expected heights 8, 9, 10, got %d, %d, %d",
			latest[0].Block.Height, latest[1].Block.Height, latest[2].Block.Height)
	}

	// Get more than available
	all := store.GetLatest(100)
	if len(all) != 10 {
		t.Errorf("Expected 10 blocks, got %d", len(all))
	}
}

func TestStore_NoDuplicates(t *testing.T) {
	store := NewStore()
	logger := zerolog.Nop()

	block := flow.BlockResult{
		Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: 1}},
		Transactions: []overflow.OverflowTransaction{{}},
		Logger:       logger,
		StartTime:    time.Now(),
	}

	// Add same block twice
	store.Add(block)
	store.Add(block)

	if store.Count() != 1 {
		t.Errorf("Expected 1 block (no duplicates), got %d", store.Count())
	}
}

func TestStore_Clear(t *testing.T) {
	store := NewStore()
	logger := zerolog.Nop()

	// Add blocks
	for i := uint64(1); i <= 5; i++ {
		store.Add(flow.BlockResult{
			Block:        flowsdk.Block{BlockHeader: flowsdk.BlockHeader{Height: i}},
			Transactions: []overflow.OverflowTransaction{{}},
			Logger:       logger,
			StartTime:    time.Now(),
		})
	}

	if store.Count() != 5 {
		t.Errorf("Expected 5 blocks, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Expected 0 blocks after clear, got %d", store.Count())
	}
}
