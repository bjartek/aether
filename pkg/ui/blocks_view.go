package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/debug"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

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
	table         table.Model
	viewport      viewport.Model
	store         *aether.Store
	keys          BlocksKeyMap
	ready         bool
	active        bool
	renderCount   int
	cachedBlocks  []flow.BlockResult
	lastBlockCount int
	maxBlocks     int // Maximum number of blocks to display
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
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
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
	return nil
}

// Update handles messages for the blocks view
func (bv *BlocksView) Update(msg tea.Msg, width, height int) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.Logger.Info().Int("width", width).Int("height", height).Msg("BlocksView WindowSizeMsg")
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
		debug.Logger.Info().Msg("Calling refreshData from WindowSizeMsg")
		bv.refreshData()
		debug.Logger.Info().Msg("refreshData completed")
	case tea.KeyMsg:
		debug.Logger.Info().Str("key", msg.String()).Msg("BlocksView KeyMsg")
		// Handle keybindings using key.Matches
		switch {
		case key.Matches(msg, bv.keys.LineDown):
			bv.table.MoveDown(1)
			return nil
		case key.Matches(msg, bv.keys.LineUp):
			bv.table.MoveUp(1)
			return nil
		case key.Matches(msg, bv.keys.GotoTop):
			bv.table.GotoTop()
			return nil
		case key.Matches(msg, bv.keys.GotoEnd):
			bv.table.GotoBottom()
			return nil
		case key.Matches(msg, bv.keys.PageDown):
			bv.table.MoveDown(bv.table.Height() / 2)
			return nil
		case key.Matches(msg, bv.keys.PageUp):
			bv.table.MoveUp(bv.table.Height() / 2)
			return nil
		}
	}

	// No viewport update needed since we're using simple text rendering
	return nil
}

// refreshData updates the table rows from the store
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

// View renders the blocks view
func (bv *BlocksView) View() string {
	if !bv.ready {
		return "Loading blocks..."
	}

	if bv.store == nil {
		return "Store not initialized..."
	}
	
	// Refresh data if needed (checks internally if refresh is necessary)
	bv.refreshData()
	
	if len(bv.cachedBlocks) == 0 {
		return "No blocks processed yet..."
	}

	// Build view from cached blocks
	var result strings.Builder
	totalBlocks := bv.store.Count()
	result.WriteString(fmt.Sprintf("Blocks with Transactions (showing %d of %d)\n\n", len(bv.cachedBlocks), totalBlocks))
	result.WriteString(fmt.Sprintf("%-15s %-15s\n", "Height", "Transactions"))
	result.WriteString(strings.Repeat("─", 30) + "\n")
	
	for _, block := range bv.cachedBlocks {
		result.WriteString(fmt.Sprintf("%-15d %-15d\n", 
			block.Block.Height,
			len(block.Transactions)))
	}
	
	return result.String()
}

// Stop is a no-op for the blocks view
func (bv *BlocksView) Stop() {
	// No cleanup needed
}
