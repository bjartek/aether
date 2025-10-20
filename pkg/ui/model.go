package ui

import (
	"fmt"
	"strings"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
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
	return NewModelWithConfig(nil)
}

// NewModelWithConfig creates a new application model with configuration.
func NewModelWithConfig(cfg *config.Config) Model {
	tabs := []Tab{
		{Name: "Dashboard", Content: ""},      // Content will be rendered by DashboardView
		{Name: "Transactions", Content: ""},   // Content will be rendered by TransactionsView
		{Name: "Events", Content: ""},         // Content will be rendered by EventsView
		{Name: "Runner", Content: ""},         // Content will be rendered by RunnerView
		{Name: "Logs", Content: ""},           // Content will be rendered by LogsView
	}

	// Configure help bubble with better visibility
	helpModel := help.New()
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7FF")) // Bright cyan for keys
	helpModel.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for descriptions
	helpModel.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")) // Gray for separators
	helpModel.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7FF")) // Bright cyan for keys
	helpModel.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for descriptions
	helpModel.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")) // Gray for separators

	// Use config if provided, otherwise use defaults
	var activeTab int
	if cfg != nil {
		switch cfg.UI.Layout.DefaultView {
		case "dashboard":
			activeTab = 0
		case "transactions":
			activeTab = 1
		case "events":
			activeTab = 2
		case "runner":
			activeTab = 3
		default:
			activeTab = 0 // Dashboard
		}
	}

	return Model{
		tabs:                tabs,
		activeTab:           activeTab,
		keys:                DefaultKeyMap(),
		help:                helpModel,
		showHelp:            false,
		dashboardView:       NewDashboardView(),
		transactionsView:    NewTransactionsViewWithConfig(cfg),
		eventsView:          NewEventsViewWithConfig(cfg),
		runnerView:          NewRunnerView(),
		logsView:            NewLogsViewWithConfig(cfg),
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
			cmd = m.logsView.Update(msg, m.width-4, contentHeight)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true

		headerHeight := 3
		helpBarHeight := 0
		if m.showHelp {
			helpBarHeight = lipgloss.Height(m.renderHelpBar())
		}
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
			cmd = m.logsView.Update(msg, m.width-4, contentHeight)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Check if we're in a text input mode where we should skip global keybindings
		inTextInput := m.isInTextInputMode()
		
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
			helpBarHeight := 0
			if m.showHelp {
				helpBarHeight = lipgloss.Height(m.renderHelpBar())
			}
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
				cmd = m.logsView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.width-4, contentHeight)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.NextTab):
			if !inTextInput {
				m.activeTab = (m.activeTab + 1) % len(m.tabs)
			}

		case key.Matches(msg, m.keys.PrevTab):
			if !inTextInput {
				m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			}
		
		// Number keys for direct tab navigation (only when not in text input)
		default:
			if !inTextInput {
				switch msg.String() {
				case "1":
					m.activeTab = m.dashboardTabIndex
					return m, nil
				case "2":
					m.activeTab = m.transactionsTabIndex
					return m, nil
				case "3":
					m.activeTab = m.eventsTabIndex
					return m, nil
				case "4":
					m.activeTab = m.runnerTabIndex
					return m, nil
				case "5":
					m.activeTab = m.logsTabIndex
					return m, nil
				}
			}
		}
	}

	// Update the appropriate view based on active tab
	headerHeight := 3
	helpBarHeight := 0
	if m.showHelp {
		helpBarHeight = lipgloss.Height(m.renderHelpBar())
	}
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
		cmd = m.logsView.Update(msg, m.width-4, contentHeight)
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
	helpBarHeight := 0
	if m.showHelp {
		helpBarHeight = lipgloss.Height(m.renderHelpBar())
	}
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
		// Add number indicator to tab name
		tabName := fmt.Sprintf("%d %s", i+1, tab.Name)
		tabs = append(tabs, style.Render(tabName))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	
	// Add help indicator with spacing
	helpIndicator := helpIndicatorStyle.Render("   ? help")
	
	headerContent := tabBar + helpIndicator
	header := headerStyle.Width(m.width).Render(headerContent)

	return header
}

// renderHelpBar renders the help bar using the help bubble.
func (m Model) renderHelpBar() string {
	if !m.showHelp {
		return ""
	}
	
	// Get the appropriate KeyMap based on active tab
	keyMap := m.getActiveKeyMap()
	
	// Use the help bubble to render
	helpView := m.help.View(keyMap)
	
	return lipgloss.NewStyle().
		Padding(0, 2, 1, 2).
		Width(m.width).
		Render(helpView)
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
		// Render logs tab - use custom style without vertical padding
		// TODO: See logs_view.go - header still scrolls when content overflows
		// This removes vertical padding to prevent adding to height,
		// and passes full contentHeight to the view (no -2 adjustment)
		logsStyle := lipgloss.NewStyle().
			Padding(0, 2). // Only horizontal padding, no vertical padding
			Width(m.width - 2)
		return logsStyle.Render(m.logsView.View())
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

// getActiveKeyMap returns the combined KeyMap for the current view.
func (m Model) getActiveKeyMap() help.KeyMap {
	// Start with global keys
	var keys CombinedKeyMap
	keys.Global = m.keys
	
	// Add view-specific keys based on active tab
	if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		keys.Transactions = &m.transactionsView.keys
	} else if m.activeTab == m.eventsTabIndex && m.eventsView != nil {
		keys.Events = &m.eventsView.keys
	} else if m.activeTab == m.runnerTabIndex && m.runnerView != nil {
		m.runnerView.mu.RLock()
		keys.Runner = &m.runnerView.keys
		m.runnerView.mu.RUnlock()
	} else if m.activeTab == m.logsTabIndex && m.logsView != nil {
		keys.Logs = &m.logsView.keys
	}
	
	return keys
}

// CombinedKeyMap combines global and view-specific key maps.
type CombinedKeyMap struct {
	Global       KeyMap
	Transactions *TransactionsKeyMap
	Events       *EventsKeyMap
	Runner       *RunnerKeyMap
	Logs         *LogsKeyMap
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k CombinedKeyMap) ShortHelp() []key.Binding {
	bindings := k.Global.ShortHelp()
	
	if k.Transactions != nil {
		bindings = append(bindings, k.Transactions.Filter, k.Transactions.ToggleFullDetail, k.Transactions.Save)
	} else if k.Events != nil {
		bindings = append(bindings, k.Events.Filter, k.Events.ToggleFullDetail)
	} else if k.Runner != nil {
		bindings = append(bindings, k.Runner.Run, k.Runner.Save, k.Runner.Refresh)
	} else if k.Logs != nil {
		bindings = append(bindings, k.Logs.Filter)
	}
	
	return bindings
}

// FullHelp returns keybindings for the expanded help view.
func (k CombinedKeyMap) FullHelp() [][]key.Binding {
	rows := k.Global.FullHelp()
	
	if k.Transactions != nil {
		rows = append(rows, []key.Binding{
			k.Transactions.LineUp, k.Transactions.LineDown,
			k.Transactions.GotoTop, k.Transactions.GotoEnd,
		})
		rows = append(rows, []key.Binding{
			k.Transactions.Filter, k.Transactions.ToggleFullDetail,
			k.Transactions.ToggleEventFields, k.Transactions.ToggleRawAddresses,
			k.Transactions.Save,
		})
	} else if k.Events != nil {
		rows = append(rows, []key.Binding{
			k.Events.LineUp, k.Events.LineDown,
			k.Events.GotoTop, k.Events.GotoEnd,
		})
		rows = append(rows, []key.Binding{
			k.Events.Filter, k.Events.ToggleFullDetail,
			k.Events.ToggleRawAddresses,
		})
	} else if k.Runner != nil {
		rows = append(rows, []key.Binding{
			k.Runner.Up, k.Runner.Down, k.Runner.Enter,
		})
		rows = append(rows, []key.Binding{
			k.Runner.Run, k.Runner.Save, k.Runner.Refresh,
		})
	} else if k.Logs != nil {
		rows = append(rows, []key.Binding{
			k.Logs.LineUp, k.Logs.LineDown,
			k.Logs.GotoTop, k.Logs.GotoEnd,
		})
		rows = append(rows, []key.Binding{
			k.Logs.Filter, k.Logs.PageDown, k.Logs.PageUp,
		})
	}
	
	return rows
}

// isInTextInputMode checks if any view is currently in text input mode
func (m Model) isInTextInputMode() bool {
	// Check transactions view filter/save mode
	if m.activeTab == m.transactionsTabIndex && m.transactionsView != nil {
		m.transactionsView.mu.RLock()
		inFilter := m.transactionsView.filterMode
		inSave := m.transactionsView.savingMode
		m.transactionsView.mu.RUnlock()
		if inFilter || inSave {
			return true
		}
	}
	
	// Check events view filter mode
	if m.activeTab == m.eventsTabIndex && m.eventsView != nil {
		m.eventsView.mu.RLock()
		inFilter := m.eventsView.filterMode
		m.eventsView.mu.RUnlock()
		if inFilter {
			return true
		}
	}
	
	// Check runner view editing/saving mode
	if m.activeTab == m.runnerTabIndex && m.runnerView != nil {
		m.runnerView.mu.RLock()
		inEdit := m.runnerView.editingField
		inSave := m.runnerView.savingConfig
		m.runnerView.mu.RUnlock()
		if inEdit || inSave {
			return true
		}
	}
	
	// Check logs view filter mode
	if m.activeTab == m.logsTabIndex && m.logsView != nil {
		if m.logsView.filterMode {
			return true
		}
	}
	
	return false
}
