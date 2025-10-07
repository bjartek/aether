package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	NextTab  key.Binding
	PrevTab  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NextTab: key.NewBinding(
			key.WithKeys("tab", "right"),
			key.WithHelp("tab/→", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab", "left"),
			key.WithHelp("shift+tab/←", "previous tab"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.PrevTab, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextTab, k.PrevTab},
		{k.Help, k.Quit},
	}
}
