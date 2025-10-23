package ui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HelpModel struct {
	help    help.Model
	ShowAll bool // Exported so parent can check if help is visible
	width   int
	keyMap  help.KeyMap
}

func NewHelpModel() HelpModel {
	h := help.New()
	h.ShowAll = true

	// Set help key/description styles using theme colors
	h.Styles.FullKey = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)
	h.Styles.FullDesc = lipgloss.NewStyle().
		Foreground(accentColor)
	h.Styles.FullSeparator = lipgloss.NewStyle().
		Foreground(mutedColor)

	return HelpModel{
		help:    h,
		ShowAll: false,
	}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) View() string {
	// Only show help when toggled on
	if !m.ShowAll {
		return ""
	}

	// Show combined keymap
	if m.keyMap == nil {
		return ""
	}

	return m.help.View(m.keyMap)
}

func (m *HelpModel) SetWidth(width int) {
	m.width = width
	m.help.Width = width
}

func (m *HelpModel) SetKeyMap(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m HelpModel) Height() int {
	if m.ShowAll {
		return lipgloss.Height(m.View())
	}
	return 0 // Invisible when help is off
}
