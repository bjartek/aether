package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/bjartek/aether/pkg/ui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tabbedModel struct {
	tabs         []*splitview.SplitViewModel
	tabNames     []string
	activeTab    int
	width        int
	height       int
	accentColor  lipgloss.Color
	mutedColor   lipgloss.Color
	pendingRows  []splitview.RowData
	footer       ui.FooterModel
	keys         tabbedKeyMap
	tabBarHeight int
}

type tabbedKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
}

func (k tabbedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.PrevTab}
}

func (k tabbedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextTab, k.PrevTab},
	}
}

func newTabbedKeyMap() tabbedKeyMap {
	return tabbedKeyMap{
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous tab"),
		),
	}
}

type loadRowsMsg struct{}

func (m tabbedModel) Init() tea.Cmd {
	// Load rows after 1 second
	return tea.Batch(
		tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
			return loadRowsMsg{}
		}),
		m.footer.Init(),
	)
}

func (m tabbedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadRowsMsg:
		// Add the pending rows to the first tab
		if len(m.pendingRows) > 0 {
			m.tabs[0].SetRows(m.pendingRows)
			m.pendingRows = nil
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.NextTab):
			// Cycle through tabs
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, nil
		case key.Matches(msg, m.keys.PrevTab):
			// Cycle backwards
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.footer.SetWidth(msg.Width)
		
		// Calculate available height for content
		tabHeight := m.tabBarHeight
		if tabHeight == 0 {
			tabHeight = ui.TabBarHeight
		}
		
		// Footer height based on whether help is shown
		footerHeight := 0
		if m.footer.ShowAll {
			footerHeight = ui.FooterHelpHeight
		}
		
		contentHeight := msg.Height - tabHeight - footerHeight
		
		// Create adjusted window size for tabs
		adjustedMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: contentHeight,
		}
		
		// Forward adjusted size to all tabs
		for i := range m.tabs {
			m.tabs[i].Update(adjustedMsg)
		}
		return m, nil
	}

	// Check if this is the help key
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "?" {
		// Toggle footer help
		oldShowAll := m.footer.ShowAll
		m.footer, _ = m.footer.Update(msg)
		
		// Adjust content height (gh-dash pattern)
		tabHeight := m.tabBarHeight
		if tabHeight == 0 {
			tabHeight = ui.TabBarHeight
		}
		
		var contentHeight int
		if m.footer.ShowAll {
			// Help shown: reduce content
			contentHeight = m.height - tabHeight - ui.FooterHelpHeight
		} else {
			// Help hidden: restore content
			contentHeight = m.height - tabHeight
		}
		
		// Update tabs with new size
		if oldShowAll != m.footer.ShowAll {
			adjustedMsg := tea.WindowSizeMsg{Width: m.width, Height: contentHeight}
			for i := range m.tabs {
				m.tabs[i].Update(adjustedMsg)
			}
		}
		
		// Don't forward ? to tabs
		return m, nil
	}

	// Forward other messages to footer
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)

	// Forward to active tab
	activeTab, cmd := m.tabs[m.activeTab].Update(msg)
	m.tabs[m.activeTab] = activeTab.(*splitview.SplitViewModel)
	
	if footerCmd != nil {
		return m, tea.Batch(cmd, footerCmd)
	}
	return m, cmd
}

func (m tabbedModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Render tab bar with clear separator
	var tabs []string
	for i, name := range m.tabNames {
		if i == m.activeTab {
			tabs = append(tabs, lipgloss.NewStyle().
				Bold(true).
				Foreground(m.accentColor).
				Padding(0, 2).
				Render(name))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(m.mutedColor).
				Padding(0, 2).
				Render(name))
		}
	}

	tabBarContent := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	tabBar := lipgloss.NewStyle().
		Background(lipgloss.Color("#073642")).
		Foreground(lipgloss.Color("#fdf6e3")).
		Width(m.width).
		Render(tabBarContent)
	
	// Cache tab bar height for layout calculations
	m.tabBarHeight = lipgloss.Height(tabBar)
	
	// Set keymaps for help
	m.footer.SetTabKeyMap(m.keys)
	m.footer.SetKeyMap(m.tabs[m.activeTab].KeyMap())
	
	// Render footer
	footerView := m.footer.View()
	
	// Use constant for footer height to avoid calculation issues
	footerHeight := 0
	if footerView != "" {
		footerHeight = ui.FooterHelpHeight
	}
	
	// Get content and constrain it to available height
	contentView := m.tabs[m.activeTab].View()
	availableContentHeight := m.height - m.tabBarHeight - footerHeight
	
	// Constrain content to available height
	if availableContentHeight > 0 {
		contentView = lipgloss.NewStyle().
			Height(availableContentHeight).
			MaxHeight(availableContentHeight).
			Render(contentView)
	}
	
	// Join: tab bar always on top, content in middle, footer at bottom (only if footer has content)
	if footerView != "" {
		return lipgloss.JoinVertical(lipgloss.Top, tabBar, contentView, footerView)
	}
	return lipgloss.JoinVertical(lipgloss.Top, tabBar, contentView)
}

func initialModel() tea.Model {
	// Read the schedule_transaction.cdc file
	codeBytes, err := os.ReadFile("example/cadence/transactions/schedule_transaction.cdc")
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %v", err))
	}
	codeOriginal := chroma.HighlightCadence(string(codeBytes))

	// Define colors for styling
	primaryColor := lipgloss.Color("#268bd2")
	accentColor := lipgloss.Color("#b58900")
	mutedColor := lipgloss.Color("#586e75")

	// Create table styles
	tableStyles := table.DefaultStyles()
	tableStyles.Header = tableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(primaryColor).
		BorderBottom(true).
		Bold(false)
	tableStyles.Selected = tableStyles.Selected.
		Foreground(lipgloss.Color("#fdf6e3")).
		Background(primaryColor).
		Bold(false)

	// Create a 256-char hex string for testing
	hexString := "a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f0a1b2c3d4e5f6071829304a5b6c7d8e9f"

	// Tab 1: Transactions
	columns1 := []splitview.ColumnConfig{
		{Name: "Name", Width: 20, SortKey: "name", FilterKey: "name"},
		{Name: "Type", Width: 15, SortKey: "type", FilterKey: "type"},
		{Name: "Network", Width: 15, SortKey: "network", FilterKey: "network"},
	}
	rows1 := []splitview.RowData{
		splitview.NewRowData(table.Row{"schedule_transaction", "Transaction", "testnet"}).
			WithCode(codeOriginal).
			WithContent(fmt.Sprintf("Code: schedule_transaction.cdc\n\nHex Data: %s", hexString)),
		splitview.NewRowData(table.Row{"transfer_tokens", "Transaction", "testnet"}).
			WithCode(codeOriginal).
			WithContent("Code: transfer_tokens.cdc"),
		splitview.NewRowData(table.Row{"update_contract", "Transaction", "mainnet"}).
			WithContent("Contract: update_contract\n\nCode not loaded."),
	}

	// Tab 2: Scripts
	columns2 := []splitview.ColumnConfig{
		{Name: "Script", Width: 25, SortKey: "name", FilterKey: "name"},
		{Name: "Network", Width: 15, SortKey: "network", FilterKey: "network"},
		{Name: "Status", Width: 10, SortKey: "status", FilterKey: "status"},
	}
	rows2 := []splitview.RowData{
		splitview.NewRowData(table.Row{"get_balance", "mainnet", "active"}).
			WithCode(codeOriginal).
			WithContent("Script: get_balance\n\nReturns account balance."),
		splitview.NewRowData(table.Row{"create_account", "emulator", "active"}).
			WithCode(codeOriginal).
			WithContent("Script: create_account"),
		splitview.NewRowData(table.Row{"list_nfts", "testnet", "inactive"}).
			WithContent("Script: list_nfts\n\nNo code available."),
	}

	return tabbedModel{
		tabs: []*splitview.SplitViewModel{
			splitview.NewSplitView(columns1, splitview.WithTableStyles(tableStyles)),
			splitview.NewSplitView(columns2, splitview.WithTableStyles(tableStyles), splitview.WithRows(rows2)),
		},
		tabNames:     []string{"Transactions", "Scripts"},
		activeTab:    0,
		accentColor:  accentColor,
		mutedColor:   mutedColor,
		pendingRows:  rows1, // Store rows to load after delay
		footer:       ui.NewFooterModel(),
		keys:         newTabbedKeyMap(),
		tabBarHeight: 0, // Will be set on first View() render
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
