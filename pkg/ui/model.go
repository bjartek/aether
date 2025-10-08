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
	tabs           []Tab
	activeTab      int
	logsTabIndex   int
	blocksTabIndex int
	logsView       *LogsView
	blocksView     *BlocksView
	help           help.Model
	keys           KeyMap
	showHelp       bool
	width          int
	height         int
	ready          bool
	helpHeight     int
}
// NewModel creates a new application model with default tabs.
func NewModel(store *aether.Store) Model {
	tabs := []Tab{
		{Name: "Blocks", Content: ""},   // Content will be rendered by BlocksView
		{Name: "Logs", Content: ""},     // Content will be rendered by LogsView
		{Name: "Settings", Content: "Settings tab content goes here."},
		{Name: "About", Content: "About this application."},
	}

	return Model{
		tabs:           tabs,
		activeTab:      1, // Start on Logs tab
		keys:           DefaultKeyMap(),
		help:           help.New(),
		showHelp:       false,
		logsView:       NewLogsView(),
		blocksView:     NewBlocksView(store),
		logsTabIndex:   1, // Index of the Logs tab
		blocksTabIndex: 0, // Index of the Blocks tab
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.logsView != nil {
		cmds = append(cmds, m.logsView.Init())
	}
	if m.blocksView != nil {
		cmds = append(cmds, m.blocksView.Init())
	}
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		// Forward tick to blocks view if it's active
		if m.blocksView != nil {
			headerHeight := 3
			footerHeight := 3
			if m.showHelp {
				footerHeight = lipgloss.Height(m.help.View(m.keys))
			}
			contentHeight := m.height - headerHeight - footerHeight
			cmd = m.blocksView.Update(msg, m.width-4, contentHeight-2)
			return m, cmd
		}
		return m, nil
	
	case logs.LogLineMsg:
		// Always forward log messages to the logs view, regardless of active tab
		if m.logsView != nil {
			headerHeight := 3
			footerHeight := 3
			if m.showHelp {
				footerHeight = lipgloss.Height(m.help.View(m.keys))
			}
			contentHeight := m.height - headerHeight - footerHeight
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
		footerHeight := 3
		if m.showHelp {
			footerHeight = lipgloss.Height(m.help.View(m.keys))
		}
		contentHeight := m.height - headerHeight - footerHeight

		// Update both views with new dimensions
		if m.logsView != nil {
			cmd = m.logsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.blocksView != nil {
			cmd = m.blocksView.Update(msg, m.width-4, contentHeight-2)
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
	footerHeight := 3
	if m.showHelp {
		footerHeight = lipgloss.Height(m.help.View(m.keys))
	}
	contentHeight := m.height - headerHeight - footerHeight

	// Update active view
	if m.activeTab == m.logsTabIndex && m.logsView != nil {
		cmd = m.logsView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.blocksTabIndex && m.blocksView != nil {
		cmd = m.blocksView.Update(msg, m.width-4, contentHeight-2)
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

	// Calculate available space for content
	headerHeight := 3
	footerHeight := 3
	if m.showHelp {
		footerHeight = m.helpHeight
	}
	contentHeight := m.height - headerHeight - footerHeight

	header := m.renderHeader()
	content := m.renderContent(contentHeight)
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the header with tab navigation.
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
	header := headerStyle.Width(m.width).Render(tabBar)

	return header
}

// renderContent renders the main content area.
func (m Model) renderContent(height int) string {
	// Render blocks tab
	if m.activeTab == m.blocksTabIndex && m.blocksView != nil {
		return contentStyle.
			Width(m.width - 2).
			Height(height).
			Render(m.blocksView.View())
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
	if m.showHelp {
		helpView = m.help.FullHelpView(m.keys.FullHelp())
	} else {
		helpView = m.help.ShortHelpView(m.keys.ShortHelp())
	}

	return footerStyle.Width(m.width).Render(helpView)
}
