package ui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FooterModel struct {
	help      help.Model
	ShowAll   bool // Exported so parent can check if help is visible
	width     int
	keyMap    help.KeyMap
	tabKeyMap help.KeyMap
	styles    FooterStyles
}

// FooterStyles defines styles for the footer container and section headers
// For help key/description colors, use WithHelpStyles option
type FooterStyles struct {
	Header  lipgloss.Style // Style for section headers ("Tab Navigation", "Component Keys")
	Content lipgloss.Style // Style for the outer container
}

type Option func(*FooterModel)

// WithHeaderStyle sets the style for section headers
func WithHeaderStyle(style lipgloss.Style) Option {
	return func(m *FooterModel) {
		m.styles.Header = style
	}
}

// WithContentStyle sets the style for help content
func WithContentStyle(style lipgloss.Style) Option {
	return func(m *FooterModel) {
		m.styles.Content = style
	}
}

// WithStyles sets both header and content styles
func WithStyles(styles FooterStyles) Option {
	return func(m *FooterModel) {
		m.styles = styles
	}
}

// WithHelpStyles sets the help bubble's key and description styles
func WithHelpStyles(keyStyle, descStyle, separatorStyle lipgloss.Style) Option {
	return func(m *FooterModel) {
		m.help.Styles.FullKey = keyStyle
		m.help.Styles.FullDesc = descStyle
		m.help.Styles.FullSeparator = separatorStyle
	}
}

func NewFooterModel(opts ...Option) FooterModel {
	h := help.New()
	h.ShowAll = true

	// Set help key/description styles
	h.Styles.FullKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true)
	h.Styles.FullDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#eee"))
	h.Styles.FullSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff"))

	m := FooterModel{
		help:    h,
		ShowAll: false,
		styles: FooterStyles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")),
			Content: lipgloss.NewStyle().
				Padding(1, 2),
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

func (m FooterModel) Init() tea.Cmd {
	return nil
}

func (m FooterModel) Update(msg tea.Msg) (FooterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check if it's the help key (?)
		if msg.String() == "?" {
			m.ShowAll = !m.ShowAll
			return m, nil
		}
	}
	return m, nil
}

func (m FooterModel) View() string {
	// Only show help when toggled on
	if !m.ShowAll {
		return ""
	}

	// Build full help overlay
	var sections []string

	// Show tab navigation keys if available
	if m.tabKeyMap != nil {
		header := m.styles.Header.Render("Tab Navigation\n")
		content := m.help.View(m.tabKeyMap)
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, header, content))
	}

	// Show component keys if available
	if m.keyMap != nil {
		header := m.styles.Header.Render("\nComponent Keys\n")
		content := m.help.View(m.keyMap)
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, header, content))
	}

	fullHelp := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return m.styles.Content.Width(m.width).Render(fullHelp)
}

func (m *FooterModel) SetWidth(width int) {
	m.width = width
	m.help.Width = width
}

func (m *FooterModel) SetKeyMap(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *FooterModel) SetTabKeyMap(keyMap help.KeyMap) {
	m.tabKeyMap = keyMap
}

func (m FooterModel) Height() int {
	if m.ShowAll {
		return lipgloss.Height(m.View())
	}
	return 0 // Invisible when help is off
}
