package tabbedtui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TabbedModel mimics cmd/layout pattern with TabbedModelPage views
type TabbedModel struct {
	tabs         []TabbedModelPage
	activeTab    int
	width        int
	height       int
	ready        bool
	help         HelpModel
	headerHeight int
	keys         TabbedModelKeyMap
	
	// Styling
	styles Styles
}

// TabbedModelKeyMap defines keybindings for tab navigation
type TabbedModelKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
	Tabs    []key.Binding
	Quit    key.Binding
	Help    key.Binding
}

func (k TabbedModelKeyMap) ShortHelp() []key.Binding {
	bindings := []key.Binding{k.NextTab, k.PrevTab}
	bindings = append(bindings, k.Tabs...)
	bindings = append(bindings, k.Quit, k.Help)
	return bindings
}

func (k TabbedModelKeyMap) FullHelp() [][]key.Binding {
	// Combine tabs and navigation in one group
	tabGroup := append([]key.Binding{k.NextTab, k.PrevTab}, k.Tabs...)
	return [][]key.Binding{
		tabGroup,
		{k.Quit, k.Help},
	}
}

// Option is a functional option for configuring TabbedModel
type Option func(*TabbedModel)

// WithStyles sets the styles for the tabbed model
func WithStyles(styles Styles) Option {
	return func(m *TabbedModel) {
		m.styles = styles
	}
}

// NewModel creates a generic tabbed model with the provided tabs.
// Views should be created externally and passed in for better composability.
func NewModel(tabs []TabbedModelPage, opts ...Option) TabbedModel {
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

	styles := NewStyles()
	helpModel := NewHelpModel()
	
	m := TabbedModel{
		tabs:      tabs,
		activeTab: 0,
		help:      helpModel,
		styles:    styles,
		keys: TabbedModelKeyMap{
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

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	// Apply help styles from the (possibly customized) styles
	m.help.SetStyles(help.Styles{
		FullKey:       m.styles.HelpKey,
		FullDesc:      m.styles.HelpDesc,
		FullSeparator: m.styles.HelpSeparator,
	})

	return m
}

// Init implements tea.Model
func (m TabbedModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, tab := range m.tabs {
		cmds = append(cmds, tab.Init())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (m TabbedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If active tab is capturing input, send ALL keys to it
		if m.tabs[m.activeTab].IsCapturingInput() {
			model, tabCmd := m.tabs[m.activeTab].Update(msg)
			m.tabs[m.activeTab] = model.(TabbedModelPage)
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
		m.tabs[m.activeTab] = model.(TabbedModelPage)

		// Return the command from the tab (don't execute it)
		// This allows async commands like ExecutionCompleteMsg to work properly
		if tabCmd != nil {
			return m, tabCmd
		}

		// Tab didn't consume it, check for help toggle
		if key.Matches(msg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
			
			// Calculate new content height based on help visibility
			// We need to render help to get its actual height
			headerHeight := m.headerHeight
			
			// Build combined keymap and render help to get actual height
			tabKeyMap := m.keys
			componentKeyMap := m.tabs[m.activeTab].KeyMap()
			combinedKeyMap := NewCombinedKeyMap(tabKeyMap, componentKeyMap)
			m.help.SetKeyMap(combinedKeyMap)
			
			// Now we can get the actual help height
			helpHeight := m.help.Height()
			
			contentHeight := m.height - headerHeight - helpHeight
			
			// Forward adjusted size to all tabs
			adjustedMsg := tea.WindowSizeMsg{
				Width:  m.width,
				Height: contentHeight,
			}
			for i := range m.tabs {
				model, _ := m.tabs[i].Update(adjustedMsg)
				m.tabs[i] = model.(TabbedModelPage)
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
			m.tabs[i] = model.(TabbedModelPage)
		}
		return m, nil

	default:
		// Broadcast all other messages to all tabs
		// Each tab decides whether to handle the message
		var cmds []tea.Cmd
		for i := range m.tabs {
			model, cmd := m.tabs[i].Update(msg)
			m.tabs[i] = model.(TabbedModelPage)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
}

// calculateContentHeight returns available height for content
func (m TabbedModel) calculateContentHeight() int {
	// Use actual help height instead of constant
	footerHeight := m.help.Height()
	return m.height - m.headerHeight - footerHeight
}

// View implements tea.Model
func (m TabbedModel) View() string {
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

	// Render help footer and calculate its actual height
	helpFooterView := m.help.View()
	helpFooterHeight := lipgloss.Height(helpFooterView)

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

// renderHeader renders tab bar
func (m TabbedModel) renderHeader() string {
	// Render all tabs
	var tabs []string
	for i, tab := range m.tabs {
		style := m.styles.Tab
		if i == m.activeTab {
			style = m.styles.ActiveTab
		}
		// Get tab name with key suffix
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
	gap := m.styles.TabGap.Render(strings.Repeat(" ", gapWidth))

	// Join tabs and gap at the bottom
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)

	// Add help indicator
	helpIndicator := m.styles.Help.Render(helpText)
	return row + helpIndicator
}

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
