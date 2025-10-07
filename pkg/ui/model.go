package ui

import (
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
	tabs         []Tab
	activeTab    int
	keys         KeyMap
	help         help.Model
	showHelp     bool
	width        int
	height       int
	ready        bool
	logsView     *LogsView
	logsTabIndex int
}

// NewModel creates a new application model with default tabs.
func NewModel() Model {
	tabs := []Tab{
		{Name: "Home", Content: "Welcome to the Home tab!"},
		{Name: "Settings", Content: "Settings tab content goes here."},
		{Name: "About", Content: "About this application."},
		{Name: "Logs", Content: ""}, // Content will be rendered by LogsView
	}

	return Model{
		tabs:         tabs,
		activeTab:    0,
		keys:         DefaultKeyMap(),
		help:         help.New(),
		showHelp:     false,
		logsView:     NewLogsView(),
		logsTabIndex: 3, // Index of the Logs tab
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	if m.logsView != nil {
		return m.logsView.Init()
	}
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true

		// Update logs view with new dimensions
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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.logsView != nil {
				m.logsView.Stop()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.NextTab):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, nil

		case key.Matches(msg, m.keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, nil
		}
	}

	// If we're on the logs tab, update the logs view
	if m.activeTab == m.logsTabIndex && m.logsView != nil {
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
		footerHeight = lipgloss.Height(m.help.View(m.keys))
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
	// If we're on the logs tab and have a logs view, render it
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
