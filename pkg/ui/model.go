package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// combinedKeyMapAdapter combines tab navigation keys and component keys
type combinedKeyMapAdapter struct {
	tabKeys       help.KeyMap
	componentKeys help.KeyMap
}

func (c combinedKeyMapAdapter) ShortHelp() []key.Binding {
	var keys []key.Binding
	if c.tabKeys != nil {
		keys = append(keys, c.tabKeys.ShortHelp()...)
	}
	if c.componentKeys != nil {
		keys = append(keys, c.componentKeys.ShortHelp()...)
	}
	return keys
}

func (c combinedKeyMapAdapter) FullHelp() [][]key.Binding {
	var groups [][]key.Binding
	if c.tabKeys != nil {
		groups = append(groups, c.tabKeys.FullHelp()...)
	}
	if c.componentKeys != nil {
		groups = append(groups, c.componentKeys.FullHelp()...)
	}
	return groups
}

// NewCombinedKeyMap creates a combined keymap from tab keys and component keys
func NewCombinedKeyMap(tabKeys, componentKeys help.KeyMap) help.KeyMap {
	return combinedKeyMapAdapter{
		tabKeys:       tabKeys,
		componentKeys: componentKeys,
	}
}

// Model mimics cmd/layout pattern with TabbedModel views
type Model struct {
	tabs               []TabbedModel
	activeTab          int
	dashboardTabIdx    int
	transactionsTabIdx int
	eventsTabIdx       int
	runnerTabIdx       int
	logsTabIdx         int
	width              int
	height             int
	ready              bool
	help               HelpModel
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

// NewModelWithConfig creates a test model with Dashboard, Transactions, Events, Runner, and Logs
func NewModelWithConfig(cfg *config.Config, debugLogger zerolog.Logger) Model {
	debugLogger.Debug().Msg("NewModelWithConfig called")

	dashboardView := NewDashboardViewWithConfig(cfg, debugLogger)
	txView := NewTransactionsViewWithConfig(cfg, debugLogger)
	eventsView := NewEventsViewWithConfig(cfg, debugLogger)
	runnerView := NewRunnerViewWithConfig(cfg, debugLogger)
	logsView := NewLogsViewWithConfig(cfg, debugLogger)
	tabs := []TabbedModel{dashboardView, txView, eventsView, runnerView, logsView}

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

	return Model{
		tabs:               tabs,
		activeTab:          0,
		dashboardTabIdx:    0,
		transactionsTabIdx: 1,
		eventsTabIdx:       2,
		runnerTabIdx:       3,
		logsTabIdx:         4,
		help:               NewHelpModel(),
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
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, tab := range m.tabs {
		cmds = append(cmds, tab.Init())
	}
	cmds = append(cmds, m.help.Init())
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		// Return the command from the tab (don't execute it)
		// This allows async commands like ExecutionCompleteMsg to work properly
		if tabCmd != nil {
			return m, tabCmd
		}

		// Tab didn't consume it, check for help toggle
		if key.Matches(msg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
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
		m.help.SetWidth(msg.Width)

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
		if txView, ok := m.tabs[m.transactionsTabIdx].(*TransactionsView); ok {
			txView.AddTransaction(msg.TransactionData)
		}
		return m, nil

	case aether.BlockEventMsg:
		// Forward event to events tab
		if eventsView, ok := m.tabs[m.eventsTabIdx].(*EventsView); ok {
			eventsView.AddEvent(msg.EventData)
		}
		return m, nil

	case logs.LogLineMsg:
		// Forward log message to logs tab only
		_, cmd := m.tabs[m.logsTabIdx].Update(msg)
		return m, cmd

	case ExecutionCompleteMsg:
		// Forward execution result to runner tab
		model, cmd := m.tabs[m.runnerTabIdx].Update(msg)
		m.tabs[m.runnerTabIdx] = model.(TabbedModel)
		return m, cmd

	case RescanFilesMsg:
		// Forward rescan message to runner tab
		model, cmd := m.tabs[m.runnerTabIdx].Update(msg)
		m.tabs[m.runnerTabIdx] = model.(TabbedModel)
		return m, cmd

	case aether.OverflowReadyMsg:
		// Set overflow and account registry in transactions and events views
		if txView, ok := m.tabs[m.transactionsTabIdx].(*TransactionsView); ok {
			txView.SetOverflow(msg.Overflow)
			txView.SetAccountRegistry(msg.AccountRegistry)
		}
		if eventsView, ok := m.tabs[m.eventsTabIdx].(*EventsView); ok {
			eventsView.SetAccountRegistry(msg.AccountRegistry)
		}
		// Set overflow and account registry in runner view
		if runnerView, ok := m.tabs[m.runnerTabIdx].(*RunnerView); ok {
			runnerView.SetOverflow(msg.Overflow)
			runnerView.SetAccountRegistry(msg.AccountRegistry)
		}
		return m, nil
	}

	// Footer doesn't handle any messages
	return m, nil
}

// calculateContentHeight returns available height for content (matching cmd/layout)
func (m Model) calculateContentHeight() int {
	headerHeight := m.headerHeight
	if headerHeight == 0 {
		headerHeight = TabBarHeight
	}
	footerHeight := 0
	if m.help.ShowAll {
		footerHeight = FooterHelpHeight
	}
	return m.height - headerHeight - footerHeight
}

// View implements tea.Model (matching cmd/layout pattern)
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render header (simple tab bar)
	header := m.renderHeader()
	m.headerHeight = lipgloss.Height(header)

	// Combine tab navigation keys and component keys into single keymap
	tabKeyMap := m.keys
	componentKeyMap := m.tabs[m.activeTab].KeyMap()
	combinedKeyMap := NewCombinedKeyMap(tabKeyMap, componentKeyMap)
	m.help.SetKeyMap(combinedKeyMap)

	// Get custom footer view from active tab (e.g., filter input)
	tabFooterView := m.tabs[m.activeTab].FooterView()
	tabFooterHeight := lipgloss.Height(tabFooterView)

	// Render help footer
	helpFooterView := m.help.View()
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

// renderHeader renders tab bar using theme styles from styles.go
func (m Model) renderHeader() string {
	// Render all tabs using theme styles
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
