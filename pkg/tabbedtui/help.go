package tabbedtui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModel wraps the bubbles help.Model to provide custom behavior
type HelpModel struct {
	help    help.Model
	ShowAll bool // Exported so parent can check if help is visible
	width   int
	keyMap  help.KeyMap
}

// NewHelpModel creates a new help model that only shows when toggled
// Styles will be set by the parent TabbedModel from its Styles configuration
func NewHelpModel() HelpModel {
	h := help.New()
	h.ShowAll = true

	return HelpModel{
		help:    h,
		ShowAll: false, // Start with help hidden
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

func (m *HelpModel) SetStyles(styles help.Styles) {
	m.help.Styles = styles
}

func (m HelpModel) Height() int {
	if m.ShowAll {
		return lipgloss.Height(m.View())
	}
	return 0 // Invisible when help is off
}
