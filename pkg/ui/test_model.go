package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestModel mimics cmd/layout pattern with TabbedModel views
type TestModel struct {
	tabs               []TabbedModel
	activeTab          int
	transactionsTabIdx int
	logsTabIdx         int
	width              int
	height             int
	ready              bool
	footer             FooterModel
	headerHeight       int
	keys               testKeyMap
}

type testKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
	Tabs    []key.Binding
	Quit    key.Binding
	Help    key.Binding
}

func (k testKeyMap) ShortHelp() []key.Binding {
	bindings := []key.Binding{k.NextTab, k.PrevTab}
	bindings = append(bindings, k.Tabs...)
	bindings = append(bindings, k.Quit, k.Help)
	return bindings
}

func (k testKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		k.Tabs,
		{k.NextTab, k.PrevTab},
		{k.Quit, k.Help},
	}
}

// NewTestModelWithConfig creates a test model with TransactionsViewV2 and LogsViewV2
func NewTestModelWithConfig(cfg *config.Config) TestModel {
	txView := NewTransactionsViewV2WithConfig(cfg)
	logsView := NewLogsViewV2WithConfig(cfg)
	tabs := []TabbedModel{txView, logsView}

	// Create tab key bindings dynamically based on number of tabs
	tabBindings := make([]key.Binding, len(tabs))
	for i := range tabs {
		keyNum := fmt.Sprintf("%d", i+1)
		helpText := fmt.Sprintf("tab: %s", tabs[i].Name())
		tabBindings[i] = key.NewBinding(
			key.WithKeys(keyNum),
			key.WithHelp(keyNum, helpText),
		)
	}

	return TestModel{
		tabs:               tabs,
		activeTab:          0,
		transactionsTabIdx: 0,
		logsTabIdx:         1,
		footer:             NewFooterModel(),
		keys: testKeyMap{
			NextTab: key.NewBinding(
				key.WithKeys("tab", "right", "l"),
				key.WithHelp("tab/→/l", "next tab"),
			),
			PrevTab: key.NewBinding(
				key.WithKeys("shift+tab", "left", "h"),
				key.WithHelp("shift+tab/←/h", "previous tab"),
			),
			Tabs: tabBindings,
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "toggle help"),
			),
		},
	}
}

// Init implements tea.Model
func (m TestModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, tab := range m.tabs {
		cmds = append(cmds, tab.Init())
	}
	cmds = append(cmds, m.footer.Init())
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (m TestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If active tab is capturing input, send ALL keys to it
		if m.tabs[m.activeTab].IsCapturingInput() {
			model, tabCmd := m.tabs[m.activeTab].Update(msg)
			m.tabs[m.activeTab] = model.(TabbedModel)
			return m, tabCmd
		}
		
		// Handle global keys first (quit, tab navigation)
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextTab):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, nil
		case key.Matches(msg, m.keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, nil
		}
		
		// Check tab number keys dynamically
		for i, tabKey := range m.keys.Tabs {
			if key.Matches(msg, tabKey) && i < len(m.tabs) {
				m.activeTab = i
				return m, nil
			}
		}
		
		// Forward to active tab - let it handle keys like "/"
		model, tabCmd := m.tabs[m.activeTab].Update(msg)
		m.tabs[m.activeTab] = model.(TabbedModel)
		
		// If tab handled it (returned a command), we're done
		if tabCmd != nil {
			return m, tabCmd
		}
		
		// Tab didn't handle it, check for help toggle
		if key.Matches(msg, m.keys.Help) {
			m.footer, _ = m.footer.Update(msg)
			// Recalculate content height after toggling help
			adjustedMsg := tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.calculateContentHeight(),
			}
			for i := range m.tabs {
				model, _ := m.tabs[i].Update(adjustedMsg)
				m.tabs[i] = model.(TabbedModel)
			}
			return m, nil
		}
		
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.footer.SetWidth(msg.Width)

		// Forward adjusted size to all tabs (matching cmd/layout pattern)
		adjustedMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: m.calculateContentHeight(),
		}
		for i := range m.tabs {
			model, _ := m.tabs[i].Update(adjustedMsg)
			m.tabs[i] = model.(TabbedModel)
		}
		return m, nil

	case aether.BlockTransactionMsg:
		// Forward transaction to transactions tab
		if txView, ok := m.tabs[m.transactionsTabIdx].(*TransactionsViewV2); ok {
			txView.AddTransaction(msg.TransactionData)
		}
		return m, nil

	case logs.LogLineMsg:
		// Forward log message to logs tab only
		_, cmd := m.tabs[m.logsTabIdx].Update(msg)
		return m, cmd
	}

	// Forward other messages to footer only (tabs already handled above)
	var footerCmd tea.Cmd
	m.footer, footerCmd = m.footer.Update(msg)
	return m, footerCmd
}

// calculateContentHeight returns available height for content (matching cmd/layout)
func (m TestModel) calculateContentHeight() int {
	headerHeight := m.headerHeight
	if headerHeight == 0 {
		headerHeight = TabBarHeight
	}
	footerHeight := 0
	if m.footer.ShowAll {
		footerHeight = FooterHelpHeight
	}
	return m.height - headerHeight - footerHeight
}

// View implements tea.Model (matching cmd/layout pattern)
func (m TestModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render header (simple tab bar)
	header := m.renderHeader()
	m.headerHeight = lipgloss.Height(header)

	// Set keymaps for help
	m.footer.SetTabKeyMap(m.keys)
	// Get keymap from active tab via TabbedModel interface
	m.footer.SetKeyMap(m.tabs[m.activeTab].KeyMap())

	// Get custom footer view from active tab (e.g., filter input)
	tabFooterView := m.tabs[m.activeTab].FooterView()
	tabFooterHeight := lipgloss.Height(tabFooterView)

	// Render help footer
	helpFooterView := m.footer.View()
	helpFooterHeight := 0
	if helpFooterView != "" {
		helpFooterHeight = FooterHelpHeight
	}

	totalFooterHeight := tabFooterHeight + helpFooterHeight

	// Get content from active tab and constrain it to available height
	contentView := m.tabs[m.activeTab].View()
	availableContentHeight := m.height - m.headerHeight - totalFooterHeight

	// Constrain content to available height
	if availableContentHeight > 0 {
		contentView = lipgloss.NewStyle().
			Height(availableContentHeight).
			MaxHeight(availableContentHeight).
			Render(contentView)
	}

	// Join: header on top, content in middle, tab footer, help footer at bottom
	parts := []string{header, contentView}
	if tabFooterView != "" {
		parts = append(parts, tabFooterView)
	}
	if helpFooterView != "" {
		parts = append(parts, helpFooterView)
	}
	return lipgloss.JoinVertical(lipgloss.Top, parts...)
}

// renderHeader renders tab bar (matching cmd/layout style exactly)
func (m TestModel) renderHeader() string {
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

	// Render all tabs
	var tabs []string
	for i, tab := range m.tabs {
		style := tabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		// Get tab name from TabbedModel interface with key suffix
		tabName := fmt.Sprintf("%s (%d)", tab.Name(), i+1)
		tabs = append(tabs, style.Render(tabName))
	}

	// Join tabs horizontally
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Calculate space for help indicator
	helpText := "? help"
	helpWidth := lipgloss.Width(helpText) + 4

	// Add gap to fill remaining space
	tabsWidth := lipgloss.Width(row)
	gapWidth := max(0, m.width-tabsWidth-helpWidth)
	gap := tabGap.Render(strings.Repeat(" ", gapWidth))

	// Join tabs and gap at the bottom
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	// Add help indicator
	helpIndicator := helpIndicatorStyle.Render(helpText)
	return row + helpIndicator
}
