package ui

import "github.com/charmbracelet/lipgloss"

// TODO: Add support for loading color schemes from ENV vars or .env file
// Example: AETHER_COLOR_SCHEME=solarized-dark|solarized-light|dracula|gruvbox
// For now, using Solarized Dark as the default theme

var (
	// Solarized Dark color palette
	// Base colors
	base03 = lipgloss.Color("#002b36") // background
	base02 = lipgloss.Color("#073642") // background highlights
	base01 = lipgloss.Color("#586e75") // comments / borders
	base0  = lipgloss.Color("#839496") // body text
	base1  = lipgloss.Color("#93a1a1") // emphasized content

	// Accent colors
	solarBlue    = lipgloss.Color("#268bd2")
	solarCyan    = lipgloss.Color("#2aa198")
	solarGreen   = lipgloss.Color("#859900")
	solarYellow  = lipgloss.Color("#b58900")
	solarOrange  = lipgloss.Color("#cb4b16")
	solarRed     = lipgloss.Color("#dc322f")
	solarMagenta = lipgloss.Color("#d33682")
	solarViolet  = lipgloss.Color("#6c71c4")

	// Semantic color mappings
	primaryColor   = solarBlue
	secondaryColor = solarCyan
	accentColor    = base1
	mutedColor     = base01
	borderColor    = base01
	successColor   = solarGreen
	errorColor     = solarRed
	warningColor   = solarOrange
	highlightColor = solarYellow

	// Base styles
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	// Header styles
	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Tab border styles
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	// Tab styles with borders
	tabStyle = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(mutedColor).
			Padding(0, 1).
			Foreground(mutedColor)

	activeTabStyle = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Foreground(primaryColor).
			Bold(true)

	tabGap = lipgloss.NewStyle().
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(true).
		BorderForeground(mutedColor)

	// Content styles
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Footer styles
	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(borderColor).
			Padding(1, 1)

	// Help indicator style
	helpIndicatorStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	// Dashboard styles
	sectionStyle = lipgloss.NewStyle().
			MarginBottom(2)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(accentColor)
)

// Exported style getters for use with tabbedtui package
func GetTabStyle() lipgloss.Style         { return tabStyle }
func GetActiveTabStyle() lipgloss.Style   { return activeTabStyle }
func GetTabGapStyle() lipgloss.Style      { return tabGap }
func GetHelpIndicatorStyle() lipgloss.Style { return helpIndicatorStyle }
