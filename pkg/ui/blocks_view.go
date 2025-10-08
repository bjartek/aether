package ui

import (
	"fmt"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(borderColor)

// TickMsg is sent periodically to refresh the blocks view
type TickMsg time.Time

// BlocksKeyMap defines keybindings for the blocks view
type BlocksKeyMap struct {
	LineUp   key.Binding
	LineDown key.Binding
	GotoTop  key.Binding
	GotoEnd  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

// DefaultBlocksKeyMap returns the default keybindings for blocks view
func DefaultBlocksKeyMap() BlocksKeyMap {
	return BlocksKeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "go to top"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "go to bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d/pgdn", "page down"),
		),
	}
}

// BlocksView manages the blocks table display
type BlocksView struct {
	table          table.Model
	viewport       viewport.Model
	store          *aether.Store
	keys           BlocksKeyMap
	ready          bool
	active         bool
	renderCount    int
	cachedBlocks   []flow.BlockResult
	lastBlockCount int
	maxBlocks      int // Maximum number of blocks to display
}

// NewBlocksView creates a new blocks view
func NewBlocksView(store *aether.Store) *BlocksView {
	columns := []table.Column{
		{Title: "Height", Width: 15},
		{Title: "Transactions", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(base03).
		Background(solarBlue).
		Bold(false)
	t.SetStyles(s)

	return &BlocksView{
		table:      t,
		store:      store,
		keys:       DefaultBlocksKeyMap(),
		ready:      false,
		maxBlocks:  100, // Limit to 100 most recent blocks
	}
}

// Init initializes the blocks view
func (bv *BlocksView) Init() tea.Cmd {
	return bv.tick()
}

// tick returns a command that sends a TickMsg after a delay
func (bv *BlocksView) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages for the blocks view
func (bv *BlocksView) Update(msg tea.Msg, width, height int) tea.Cmd {
	switch msg := msg.(type) {
	case TickMsg:
		// Refresh data on tick
		bv.refreshData()
		return bv.tick()
	
	case tea.WindowSizeMsg:
		if !bv.ready {
			bv.viewport = viewport.New(width, height)
			bv.viewport.KeyMap = viewport.KeyMap{
				PageDown: bv.keys.PageDown,
				PageUp:   bv.keys.PageUp,
				Down:     bv.keys.LineDown,
				Up:       bv.keys.LineUp,
			}
			bv.ready = true
		} else {
			bv.viewport.Width = width
			bv.viewport.Height = height
		}
		bv.table.SetWidth(width)
		bv.table.SetHeight(height)
		// Refresh data when window is resized
		bv.refreshData()
	case tea.KeyMsg:
		// Let the table handle the key - no data refresh needed
		var cmd tea.Cmd
		bv.table, cmd = bv.table.Update(msg)
		return cmd
	}

	// No viewport update needed since we're using simple text rendering
	return nil
}

// refreshData updates the table rows from the store
// Should only be called from Update(), never from View()
func (bv *BlocksView) refreshData() {
	if bv.store == nil {
		return
	}

	// Check if we need to refresh by comparing block count
	currentCount := bv.store.Count()
	if currentCount == bv.lastBlockCount {
		return // No new blocks
	}
	
	// Get only the latest N blocks to avoid copying too much data
	bv.cachedBlocks = bv.store.GetLatest(bv.maxBlocks)
	bv.lastBlockCount = currentCount
	
	// Build table rows from cached blocks
	rows := make([]table.Row, len(bv.cachedBlocks))
	for i, block := range bv.cachedBlocks {
		rows[i] = table.Row{
			fmt.Sprintf("%d", block.Block.Height),
			fmt.Sprintf("%d", len(block.Transactions)),
		}
	}

	bv.table.SetRows(rows)
}

// View renders the blocks view - PURE FUNCTION, no state mutations
func (bv *BlocksView) View() string {
	if !bv.ready {
		return "Loading blocks..."
	}

	if bv.store == nil {
		return "Store not initialized..."
	}
	
	if len(bv.cachedBlocks) == 0 {
		return "No blocks processed yet..."
	}

	// Render header with count
	totalBlocks := bv.store.Count()
	header := fmt.Sprintf("Blocks with Transactions (showing %d of %d)\n\n", len(bv.cachedBlocks), totalBlocks)
	
	// Render the table directly (no extra styling wrapper)
	return header + bv.table.View()
}

// Stop is a no-op for the blocks view
func (bv *BlocksView) Stop() {
	// No cleanup needed
}
