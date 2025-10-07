package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#EE6FF8")
	accentColor    = lipgloss.Color("#FFFDF5")
	mutedColor     = lipgloss.Color("#626262")
	borderColor    = lipgloss.Color("#383838")

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

	// Tab styles
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(mutedColor)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(accentColor).
			Background(primaryColor).
			Bold(true)

	// Content styles
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			BorderLeft(true).
			BorderRight(true)

	// Footer styles
	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(borderColor).
			Padding(1, 1)
)
