package ui

import (
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/bjartek/overflow/v2"
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
	eventsTabIndex      int
	runnerTabIndex      int
	logsTabIndex        int
	dashboardView       *DashboardView
	transactionsView    *TransactionsView
	eventsView          *EventsView
	runnerView          *RunnerView
	logsView            *LogsView
	help                help.Model
	keys                KeyMap
	showHelp            bool
	width               int
	height              int
	ready               bool
}

// NewModel creates a new application model with default tabs.
func NewModel() Model {
	tabs := []Tab{
		{Name: "Dashboard", Content: ""},      // Content will be rendered by DashboardView
		{Name: "Transactions", Content: ""},   // Content will be rendered by TransactionsView
		{Name: "Events", Content: ""},         // Content will be rendered by EventsView
		{Name: "Runner", Content: ""},         // Content will be rendered by RunnerView
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
		eventsView:          NewEventsView(),
		runnerView:          NewRunnerView(),
		logsView:            NewLogsView(),
		dashboardTabIndex:   0, // Index of the Dashboard tab
		transactionsTabIndex: 1, // Index of the Transactions tab
		eventsTabIndex:      2, // Index of the Events tab
		runnerTabIndex:      3, // Index of the Runner tab
		logsTabIndex:        4, // Index of the Logs tab
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
	if m.eventsView != nil {
		cmds = append(cmds, m.eventsView.Init())
	}
	if m.runnerView != nil {
		cmds = append(cmds, m.runnerView.Init())
	}
	if m.logsView != nil {
		cmds = append(cmds, m.logsView.Init())
	}
	return tea.Batch(cmds...)
}

// SetOverflow sets the overflow instance for the runner view
func (m *Model) SetOverflow(o *overflow.OverflowState) {
	if m.runnerView != nil {
		m.runnerView.SetOverflow(o)
	}
}

// SetAccountRegistry sets the account registry for views that need it
func (m *Model) SetAccountRegistry(registry *aether.AccountRegistry) {
	if m.runnerView != nil {
		m.runnerView.SetAccountRegistry(registry)
	}
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case aether.OverflowReadyMsg:
		// Set overflow and account registry in runner view
		m.SetOverflow(msg.Overflow)
		m.SetAccountRegistry(msg.AccountRegistry)
		return m, nil

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
		
		// Forward events from transaction to events view
		if m.eventsView != nil && len(msg.Transaction.Events) > 0 {
			for eventIndex, event := range msg.Transaction.Events {
				m.eventsView.AddEvent(
					msg.BlockHeight,
					msg.BlockID,
					msg.Transaction.Id,
					msg.Transaction.TransactionIndex,
					event,
					eventIndex,
					msg.AccountRegistry,
				)
			}
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
		helpBarHeight := m.calculateHelpBarHeight()
		contentHeight := m.height - headerHeight - helpBarHeight

		// Update all views with new dimensions
		if m.dashboardView != nil {
			cmd = m.dashboardView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.transactionsView != nil {
			cmd = m.transactionsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.eventsView != nil {
			cmd = m.eventsView.Update(msg, m.width-4, contentHeight-2)
			cmds = append(cmds, cmd)
		}
		if m.runnerView != nil {
			cmd = m.runnerView.Update(msg, m.width-4, contentHeight-2)
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
			// Need to update all views with new content height after toggling help
			headerHeight := 3
			helpBarHeight := m.calculateHelpBarHeight()
			contentHeight := m.height - headerHeight - helpBarHeight
			
			// Update all views with new dimensions
			if m.dashboardView != nil {
				cmd = m.dashboardView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight-2)
				cmds = append(cmds, cmd)
			}
			if m.transactionsView != nil {
				cmd = m.transactionsView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight-2)
				cmds = append(cmds, cmd)
			}
			if m.eventsView != nil {
				cmd = m.eventsView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight-2)
				cmds = append(cmds, cmd)
			}
			if m.runnerView != nil {
				cmd = m.runnerView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight-2)
				cmds = append(cmds, cmd)
			}
			if m.logsView != nil {
				cmd = m.logsView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight-2)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.NextTab):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)

		case key.Matches(msg, m.keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		}
	}

	// Update the appropriate view based on active tab
	headerHeight := 3
	helpBarHeight := m.calculateHelpBarHeight()
	contentHeight := m.height - headerHeight - helpBarHeight

	// Update active view
	if m.activeTab == m.dashboardTabIndex && m.dashboardView != nil {
		cmd = m.dashboardView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		cmd = m.transactionsView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.eventsTabIndex && m.eventsView != nil {
		cmd = m.eventsView.Update(msg, m.width-4, contentHeight-2)
		cmds = append(cmds, cmd)
	} else if m.activeTab == m.runnerTabIndex && m.runnerView != nil {
		cmd = m.runnerView.Update(msg, m.width-4, contentHeight-2)
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

// calculateHelpBarHeight calculates the height of the help bar.
func (m Model) calculateHelpBarHeight() int {
	if !m.showHelp {
		return 0
	}
	helpText := m.getHelpText()
	maxWidth := m.width - 10
	wrappedHelp := wrapText(helpText, maxWidth)
	return strings.Count(wrappedHelp, "\n") + 2 // +2 for spacing
}

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate available space for content
	headerHeight := 3
	helpBarHeight := m.calculateHelpBarHeight()
	contentHeight := m.height - headerHeight - helpBarHeight

	header := m.renderHeader()
	helpBar := m.renderHelpBar()
	content := m.renderContent(contentHeight)

	if m.showHelp {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			helpBar,
			content,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
	)
}

// renderHeader renders the header with tab navigation and help indicator.
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
	
	// Add help indicator with spacing
	helpIndicator := helpIndicatorStyle.Render("   ? help")
	
	headerContent := tabBar + helpIndicator
	header := headerStyle.Width(m.width).Render(headerContent)

	return header
}

// renderHelpBar renders the help bar as a separate section.
func (m Model) renderHelpBar() string {
	if !m.showHelp {
		return ""
	}
	
	helpText := m.getHelpText()
	
	// Wrap help text to available width with proper word wrapping
	maxWidth := m.width - 10 // Leave some margin
	wrappedHelp := wrapText(helpText, maxWidth)
	
	helpContent := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("Help: ") + 
		lipgloss.NewStyle().
			Foreground(mutedColor).
			Render(wrappedHelp)
	
	// Render with padding and width
	return lipgloss.NewStyle().
		Padding(0, 2, 1, 2). // top, right, bottom, left
		Width(m.width).
		Render(helpContent)
}

// renderContent renders the main content area.
func (m Model) renderContent(height int) string {
	var viewContent string
	
	// Render dashboard tab
	if m.activeTab == m.dashboardTabIndex && m.dashboardView != nil {
		viewContent = m.dashboardView.View()
	} else if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		// Render transactions tab
		viewContent = m.transactionsView.View()
	} else if m.activeTab == m.eventsTabIndex && m.eventsView != nil {
		// Render events tab
		viewContent = m.eventsView.View()
	} else if m.activeTab == m.runnerTabIndex && m.runnerView != nil {
		// Render runner tab
		viewContent = m.runnerView.View()
	} else if m.activeTab == m.logsTabIndex && m.logsView != nil {
		// Render logs tab
		viewContent = m.logsView.View()
	} else {
		// Otherwise render static content
		viewContent = m.tabs[m.activeTab].Content
	}
	
	return contentStyle.
		Width(m.width - 2).
		Height(height).
		Render(viewContent)
}

// wrapText wraps text to fit within maxWidth, breaking at word boundaries.
func wrapText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	
	// Split by bullet points to keep them together
	parts := strings.Split(text, " • ")
	var lines []string
	var currentLine string
	
	for i, part := range parts {
		// Add bullet back except for first item
		itemText := part
		if i > 0 {
			itemText = "• " + part
		}
		
		// Check if adding this item would exceed width
		if currentLine == "" {
			currentLine = itemText
		} else if len(currentLine) + 3 + len(part) <= maxWidth {
			currentLine += " • " + part
		} else {
			// Current line is full, start new line
			lines = append(lines, currentLine)
			currentLine = itemText
		}
	}
	
	// Add the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	// Indent continuation lines for better readability
	for i := 1; i < len(lines); i++ {
		lines[i] = "      " + lines[i]
	}
	
	return strings.Join(lines, "\n")
}

// getHelpText returns the help text for the current view.
func (m Model) getHelpText() string {
	if !m.showHelp {
		return ""
	}

	var helpItems []string
	
	// Global keybindings
	helpItems = append(helpItems, "tab/→: next")
	helpItems = append(helpItems, "shift+tab/←: prev")
	helpItems = append(helpItems, "q: quit")
	
	// View-specific keybindings
	if m.activeTab == m.transactionsTabIndex {
		helpItems = append(helpItems, "/: filter")
		helpItems = append(helpItems, "j/k: navigate")
		helpItems = append(helpItems, "enter/d: detail")
		helpItems = append(helpItems, "e: events")
		helpItems = append(helpItems, "a: addresses")
		helpItems = append(helpItems, "g/G: top/bottom")
	} else if m.activeTab == m.eventsTabIndex {
		helpItems = append(helpItems, "/: filter")
		helpItems = append(helpItems, "j/k: navigate")
		helpItems = append(helpItems, "enter/d: detail")
		helpItems = append(helpItems, "a: addresses")
		helpItems = append(helpItems, "g/G: top/bottom")
	} else if m.activeTab == m.runnerTabIndex {
		helpItems = append(helpItems, "j/k: navigate")
		helpItems = append(helpItems, "enter: edit field")
		helpItems = append(helpItems, "esc: stop editing")
		helpItems = append(helpItems, "r: run")
	} else if m.activeTab == m.logsTabIndex {
		helpItems = append(helpItems, "/: filter")
		helpItems = append(helpItems, "j/k: scroll")
		helpItems = append(helpItems, "g/G: top/bottom")
		helpItems = append(helpItems, "ctrl+u/d: page")
	}
	
	return strings.Join(helpItems, " • ")
}
