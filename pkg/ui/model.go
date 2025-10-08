package ui

import (
	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab in the application.
type Tab struct {
	Name    string
	Content string
}

// Model represents the application state.
type Model struct {
	tabs                []Tab
	activeTab           int
	dashboardTabIndex   int
	transactionsTabIndex int
	logsTabIndex        int
	dashboardView       *DashboardView
	transactionsView    *TransactionsView
	logsView            *LogsView
	help                help.Model
	keys                KeyMap
	showHelp            bool
	width               int
	height              int
	ready               bool
	helpHeight          int
}

// NewModel creates a new application model with default tabs.
func NewModel() Model {
	tabs := []Tab{
		{Name: "Dashboard", Content: ""},      // Content will be rendered by DashboardView
		{Name: "Transactions", Content: ""},   // Content will be rendered by TransactionsView
		{Name: "Logs", Content: ""},           // Content will be rendered by LogsView
	}

	return Model{
		tabs:                tabs,
		activeTab:           0, // Start on Dashboard tab
		keys:                DefaultKeyMap(),
		help:                help.New(),
		showHelp:            false,
		dashboardView:       NewDashboardView(),
		transactionsView:    NewTransactionsView(),
		logsView:            NewLogsView(),
		dashboardTabIndex:   0, // Index of the Dashboard tab
		transactionsTabIndex: 1, // Index of the Transactions tab
		logsTabIndex:        2, // Index of the Logs tab
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.dashboardView != nil {
		cmds = append(cmds, m.dashboardView.Init())
	}
	if m.transactionsView != nil {
		cmds = append(cmds, m.transactionsView.Init())
	}
	if m.logsView != nil {
		cmds = append(cmds, m.logsView.Init())
	}
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case aether.BlockTransactionMsg:
		// Forward transaction messages to the transactions view
		if m.transactionsView != nil {
			m.transactionsView.AddTransaction(
				msg.BlockHeight,
				msg.BlockID,
				msg.Transaction,
				msg.AccountRegistry,
			)
		}
		return m, nil
	
	case logs.LogLineMsg:
		// Always forward log messages to the logs view, regardless of active tab
		if m.logsView != nil {
			headerHeight := 3
			contentHeight := m.height - headerHeight
			cmd = m.logsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true

		headerHeight := 3
		contentHeight := m.height - headerHeight

		// Update all views with new dimensions
		if m.dashboardView != nil {
			cmd = m.dashboardView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.transactionsView != nil {
			cmd = m.transactionsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.logsView != nil {
			cmd = m.logsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.logsView != nil {
				m.logsView.Stop()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.helpHeight = lipgloss.Height(m.help.View(m.keys))
			}
			return m, nil

		case key.Matches(msg, m.keys.NextTab):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)

		case key.Matches(msg, m.keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		}
	}

	// Update the appropriate view based on active tab
	headerHeight := 3
	contentHeight := m.height - headerHeight

	// Update active view
	if m.activeTab == m.dashboardTabIndex && m.dashboardView != nil {
		cmd = m.dashboardView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		cmd = m.transactionsView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.logsTabIndex && m.logsView != nil {
		cmd = m.logsView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	}

	// Filter out nil commands
	var validCmds []tea.Cmd
	for _, c := range cmds {
		if c != nil {
			validCmds = append(validCmds, c)
		}
	}
	
	if len(validCmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(validCmds...)
}

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate available space for content (no footer)
	headerHeight := 3
	contentHeight := m.height - headerHeight

	header := m.renderHeader()
	content := m.renderContent(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
	)
}

// renderHeader renders the header with tab navigation and hints.
func (m Model) renderHeader() string {
	var tabs []string
	for i, tab := range m.tabs {
		style := tabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		tabs = append(tabs, style.Render(tab.Name))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	
	// Add tab-specific help hints
	hints := ""
	if m.activeTab == m.transactionsTabIndex {
		hints = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(" • /: filter • j/k: navigate • enter/d: detail • e: events • a: addresses • g/G: top/bottom")
	} else if m.activeTab == m.logsTabIndex {
		hints = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(" • /: filter • j/k: scroll • g/G: top/bottom • ctrl+u/d: page")
	}
	
	headerContent := tabBar + hints
	header := headerStyle.Width(m.width).Render(headerContent)

	return header
}

// renderContent renders the main content area.
func (m Model) renderContent(height int) string {
	// Render dashboard tab
	if m.activeTab == m.dashboardTabIndex && m.dashboardView != nil {
		return contentStyle.
			Width(m.width - 2).
			Height(height).
			Render(m.dashboardView.View())
	}

	// Render transactions tab
	if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		return contentStyle.
			Width(m.width - 2).
			Height(height).
			Render(m.transactionsView.View())
	}

	// Render logs tab
	if m.activeTab == m.logsTabIndex && m.logsView != nil {
		return contentStyle.
			Width(m.width - 2).
			Height(height).
			Render(m.logsView.View())
	}

	// Otherwise render static content
	content := m.tabs[m.activeTab].Content

	return contentStyle.
		Width(m.width - 2).
		Height(height).
		Render(content)
}

// renderFooter renders the footer with help information.
func (m Model) renderFooter() string {
	var helpView string
	
	// Add tab-specific help hints
	tabHints := ""
	if m.activeTab == m.transactionsTabIndex {
		tabHints = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("  [j/k: navigate • enter/d: toggle detail • e: event fields • a: raw addresses • g/G: top/bottom]")
	} else if m.activeTab == m.logsTabIndex {
		tabHints = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("  [j/k: scroll • g/G: top/bottom • ctrl+u/d: page up/down]")
	}
	
	if m.showHelp {
		helpView = m.help.FullHelpView(m.keys.FullHelp())
	} else {
		helpView = m.help.ShortHelpView(m.keys.ShortHelp())
	}

	return footerStyle.Width(m.width).Render(helpView + tabHints)
}
