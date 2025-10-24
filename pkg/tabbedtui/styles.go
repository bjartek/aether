package tabbedtui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the styling for the tabbed model
type Styles struct {
	Tab            lipgloss.Style
	ActiveTab      lipgloss.Style
	TabGap         lipgloss.Style
	Help           lipgloss.Style
	HelpKey        lipgloss.Style
	HelpDesc       lipgloss.Style
	HelpSeparator  lipgloss.Style
}

// StyleOption is a functional option for configuring Styles
type StyleOption func(*Styles)

// WithPrimaryColor sets the primary color for active tabs
func WithPrimaryColor(color lipgloss.Color) StyleOption {
	return func(s *Styles) {
		s.ActiveTab = s.ActiveTab.
			BorderForeground(color).
			Foreground(color)
	}
}

// WithMutedColor sets the muted color for inactive tabs and borders
func WithMutedColor(color lipgloss.Color) StyleOption {
	return func(s *Styles) {
		s.Tab = s.Tab.
			BorderForeground(color).
			Foreground(color)
		s.TabGap = s.TabGap.
			BorderForeground(color)
		s.Help = s.Help.
			Foreground(color)
	}
}

// NewStyles creates a new Styles with optional color customization
func NewStyles(opts ...StyleOption) Styles {
	// Default colors (Solarized Dark inspired)
	primaryColor := lipgloss.Color("#268bd2")
	accentColor := lipgloss.Color("#93a1a1")
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

	s := Styles{
		Tab: lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(mutedColor).
			Padding(0, 1).
			Foreground(mutedColor),
		ActiveTab: lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Foreground(primaryColor).
			Bold(true),
		TabGap: lipgloss.NewStyle().
			BorderTop(false).
			BorderLeft(false).
			BorderRight(false).
			BorderBottom(true).
			BorderForeground(mutedColor),
		Help: lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 2),
		HelpKey: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true),
		HelpDesc: lipgloss.NewStyle().
			Foreground(accentColor),
		HelpSeparator: lipgloss.NewStyle().
			Foreground(mutedColor),
	}

	// Apply options
	for _, opt := range opts {
		opt(&s)
	}

	return s
}
