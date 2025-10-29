package ui

import (
	"github.com/bjartek/aether/pkg/tabbedtui"
	"github.com/charmbracelet/lipgloss"
)

// TODO: Add support for loading color schemes from ENV vars or .env file
// Example: AETHER_COLOR_SCHEME=solarized-dark|solarized-light|dracula|gruvbox
// For now, using Solarized Dark as the default theme

var (
	// Solarized Dark color palette
	// Base colors
	base03 = lipgloss.Color("#002b36") // background
	base02 = lipgloss.Color("#073642") // background highlights
	base01 = lipgloss.Color("#586e75") // comments / borders
	base1  = lipgloss.Color("#93a1a1") // emphasized content

	// Accent colors
	solarBlue   = lipgloss.Color("#268bd2")
	solarCyan   = lipgloss.Color("#2aa198")
	solarGreen  = lipgloss.Color("#859900")
	solarYellow = lipgloss.Color("#b58900")
	solarRed    = lipgloss.Color("#dc322f")

	// Semantic color mappings
	primaryColor   = solarBlue
	secondaryColor = solarCyan
	tertiaryColor  = solarGreen
	accentColor    = base1
	mutedColor     = base01
	borderColor    = base01
	successColor   = solarGreen
	errorColor     = solarRed
	highlightColor = solarYellow

	labelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(secondaryColor)

	valueStyle = lipgloss.NewStyle().
		Foreground(accentColor)

	dimStyle = lipgloss.NewStyle().
		Foreground(mutedColor)
)

// GetTabbedStyles returns the tabbedtui.Styles configured with our theme colors
func GetTabbedStyles() tabbedtui.Styles {
	return tabbedtui.NewStyles(
		tabbedtui.WithPrimaryColor(primaryColor),
		tabbedtui.WithMutedColor(mutedColor),
	)
}
