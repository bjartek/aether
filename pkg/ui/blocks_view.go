package ui

import (
	"fmt"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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
	table table.Model
	store *aether.Store
	keys  BlocksKeyMap
	ready bool
}

// NewBlocksView creates a new blocks view
func NewBlocksView(store *aether.Store) *BlocksView {
	columns := []table.Column{
		{Title: "Height", Width: 10},
		{Title: "Transactions", Width: 12},
		{Title: "System Tx", Width: 10},
		{Title: "Events", Width: 10},
		{Title: "Duration", Width: 12},
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
		table: t,
		store: store,
		keys:  DefaultBlocksKeyMap(),
		ready: false,
	}
}

// Init initializes the blocks view
func (bv *BlocksView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the blocks view
func (bv *BlocksView) Update(msg tea.Msg, width, height int) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !bv.ready {
			bv.table.SetWidth(width)
			bv.table.SetHeight(height)
			bv.ready = true
		} else {
			bv.table.SetWidth(width)
			bv.table.SetHeight(height)
		}
	case tea.KeyMsg:
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

	// Update table data from store
	bv.refreshData()

	// Update table (handles scrolling with arrow keys)
	bv.table, cmd = bv.table.Update(msg)
	return cmd
}

// refreshData updates the table rows from the store
func (bv *BlocksView) refreshData() {
	if bv.store == nil {
		return
	}

	blocks := bv.store.GetAll()
	rows := make([]table.Row, len(blocks))

	for i, block := range blocks {
		duration := time.Since(block.StartTime)
		rows[i] = table.Row{
			fmt.Sprintf("%d", block.Block.Height),
			fmt.Sprintf("%d", len(block.Transactions)),
			fmt.Sprintf("%d", len(block.SystemTransactions)),
			fmt.Sprintf("%d", len(block.SystemChunkEvents)),
			fmt.Sprintf("%dms", duration.Milliseconds()),
		}
	}

	bv.table.SetRows(rows)
}

// View renders the blocks view
func (bv *BlocksView) View() string {
	if !bv.ready {
		return "Loading blocks..."
	}

	if bv.store == nil || bv.store.Count() == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Render("No blocks processed yet...")
	}

	return tableStyle.Render(bv.table.View())
}

// Stop is a no-op for the blocks view
func (bv *BlocksView) Stop() {
	// No cleanup needed
}
