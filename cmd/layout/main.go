package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bjartek/aether/pkg/chroma"
	"github.com/bjartek/aether/pkg/splitview"
	"github.com/bjartek/aether/pkg/ui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dashboardWrapper wraps DashboardView to implement tea.Model
type dashboardWrapper struct {
	dashboard *ui.DashboardView
	width     int
	height    int
}

func (d dashboardWrapper) Init() tea.Cmd {
	return d.dashboard.Init()
}

func (d dashboardWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size messages
	if windowMsg, ok := msg.(tea.WindowSizeMsg); ok {
		d.width = windowMsg.Width
		d.height = windowMsg.Height
	}
	
	cmd := d.dashboard.Update(msg, d.width, d.height)
	return d, cmd
}

func (d dashboardWrapper) View() string {
	return d.dashboard.View()
}

type tabbedModel struct {
	tabs         []tea.Model
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
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
}

func (k tabbedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.PrevTab, k.Tab1, k.Tab2}
}

func (k tabbedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
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
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "tab 1"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "tab 2"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "tab 3"),
		),
		Tab4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "tab 4"),
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
			// Only set rows if the tab is a SplitViewModel
			if splitView, ok := m.tabs[0].(*splitview.SplitViewModel); ok {
				splitView.SetRows(m.pendingRows)
			}
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

		// Handle number key presses for direct tab access
		if key.Matches(msg, m.keys.Tab1) && len(m.tabs) >= 1 {
			m.activeTab = 0
			return m, nil
		}
		if key.Matches(msg, m.keys.Tab2) && len(m.tabs) >= 2 {
			m.activeTab = 1
			return m, nil
		}
		if key.Matches(msg, m.keys.Tab3) && len(m.tabs) >= 3 {
			m.activeTab = 2
			return m, nil
		}
		if key.Matches(msg, m.keys.Tab4) && len(m.tabs) >= 4 {
			m.activeTab = 3
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
			m.tabs[i], _ = m.tabs[i].Update(adjustedMsg)
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
				m.tabs[i], _ = m.tabs[i].Update(adjustedMsg)
			}
		}
		
		// Don't forward ? to tabs
		return m, nil
	}

	// Forward other messages to footer
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)

	// Forward to active tab
	var cmd tea.Cmd
	m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
	
	if footerCmd != nil {
		return m, tea.Batch(cmd, footerCmd)
	}
	return m, cmd
}

func (m tabbedModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Render tab bar with fancy borders (from ui/model.go)
	primaryColor := lipgloss.Color("#268bd2")
	mutedColor := lipgloss.Color("#586e75")
	
	// Tab border styles
	activeTabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tabStyle := lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Foreground(mutedColor)

	activeTabStyle := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(primaryColor).
		Padding(0, 1).
		Foreground(primaryColor).
		Bold(true)

	tabGap := lipgloss.NewStyle().
		BorderTop(true).
		BorderBottom(true).
		BorderForeground(mutedColor)

	helpIndicatorStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		BorderTop(true).
		BorderForeground(mutedColor).
		Padding(0, 1)

	var tabs []string
	for i, name := range m.tabNames {
		style := tabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		// Add number indicator to tab name
		tabName := fmt.Sprintf("%d %s", i+1, name)
		tabs = append(tabs, style.Render(tabName))
	}

	// Join tabs horizontally
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Calculate space needed for help indicator
	helpText := "? help"
	helpWidth := lipgloss.Width(helpText) + 4 // +4 for padding

	// Add gap to fill remaining space with bottom border
	tabsWidth := lipgloss.Width(row)
	gapWidth := max(0, m.width-tabsWidth-helpWidth)
	gap := tabGap.Render(strings.Repeat(" ", gapWidth))

	// Join tabs and gap at the bottom (so gap's bottom border aligns)
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	// Add help indicator at the end
	helpIndicator := helpIndicatorStyle.Render(helpText)
	tabBar := row + helpIndicator
	
	// Cache tab bar height for layout calculations
	m.tabBarHeight = lipgloss.Height(tabBar)
	
	// Set keymaps for help
	m.footer.SetTabKeyMap(m.keys)
	// Only set keymap if the tab supports it (e.g., SplitViewModel)
	if splitView, ok := m.tabs[m.activeTab].(*splitview.SplitViewModel); ok {
		m.footer.SetKeyMap(splitView.KeyMap())
	}
	
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
		tabs: []tea.Model{
			splitview.NewSplitView(columns1, splitview.WithTableStyles(tableStyles)),
			splitview.NewSplitView(columns2, splitview.WithTableStyles(tableStyles), splitview.WithRows(rows2)),
			dashboardWrapper{dashboard: ui.NewDashboardView()},
		},
		tabNames:     []string{"Transactions", "Scripts", "Dashboard"},
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
