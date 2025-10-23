package ui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

// InputHandledMsg signals that a tab has consumed a key input
// and parent should not process it further
type InputHandledMsg struct{}

// InputHandled returns a command that signals input was handled
func InputHandled() tea.Cmd {
	return func() tea.Msg { return InputHandledMsg{} }
}

// TabbedModel defines the interface for models that can be used as tabs
type TabbedModel interface {
	tea.Model
	
	// Name returns the display name for this tab
	Name() string
	
	// KeyMap returns the key bindings for this tab (for help display)
	KeyMap() help.KeyMap
	
	// FooterView returns optional footer content for this tab
	// Can be status text, interactive UI, or empty string
	// This is rendered ABOVE the help footer
	FooterView() string
	
	// IsCapturingInput returns true if the tab is capturing input
	// (e.g., filter mode, form input) and should receive ALL keys
	// When true, parent should skip global key handling
	IsCapturingInput() bool
}
