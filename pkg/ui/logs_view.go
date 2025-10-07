package ui

import (
	"strings"

	"github.com/bjartek/aether/pkg/logs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsKeyMap defines keybindings for the logs view
type LogsKeyMap struct {
	LineUp   key.Binding
	LineDown key.Binding
	GotoTop  key.Binding
	GotoEnd  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

// DefaultLogsKeyMap returns the default keybindings for logs view
func DefaultLogsKeyMap() LogsKeyMap {
	return LogsKeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "go to top"),
		),
		GotoEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "go to bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d/pgdn", "page down"),
		),
	}
}

// LogsView manages the logs display with a scrollable viewport.
type LogsView struct {
	viewport viewport.Model
	lines    []string
	maxLines int
	keys     LogsKeyMap
	ready    bool
}

// NewLogsView creates a new logs view.
func NewLogsView() *LogsView {
	return &LogsView{
		lines:    make([]string, 0),
		maxLines: 1000, // Keep last 1000 lines in memory
		keys:     DefaultLogsKeyMap(),
		ready:    false,
	}
}

// Init initializes the logs view.
func (lv *LogsView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the logs view.
func (lv *LogsView) Update(msg tea.Msg, width, height int) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case logs.LogLineMsg:
		if msg.Err != nil {
			// Handle error - could add error display
			return nil
		}

		if msg.Line != "" {
			// Check if we're at the bottom before adding new content
			atBottom := false
			if lv.ready {
				atBottom = lv.viewport.AtBottom()
			}

			// Add new line
			lv.lines = append(lv.lines, strings.TrimRight(msg.Line, "\n"))

			// Keep only the last maxLines
			if len(lv.lines) > lv.maxLines {
				lv.lines = lv.lines[len(lv.lines)-lv.maxLines:]
			}

			// Update viewport content
			if lv.ready {
				lv.viewport.SetContent(strings.Join(lv.lines, "\n"))
				// Only auto-scroll if we were at the bottom
				if atBottom {
					lv.viewport.GotoBottom()
				}
			}
		}

		return nil

	case tea.WindowSizeMsg:
		if !lv.ready {
			lv.viewport = viewport.New(width, height)
			lv.viewport.SetContent(strings.Join(lv.lines, "\n"))
			lv.ready = true
		} else {
			lv.viewport.Width = width
			lv.viewport.Height = height
		}

	case tea.KeyMsg:
		// Handle keybindings using key.Matches
		switch {
		case key.Matches(msg, lv.keys.LineDown):
			lv.viewport.ViewDown()
			return nil
		case key.Matches(msg, lv.keys.LineUp):
			lv.viewport.ViewUp()
			return nil
		case key.Matches(msg, lv.keys.GotoTop):
			lv.viewport.GotoTop()
			return nil
		case key.Matches(msg, lv.keys.GotoEnd):
			lv.viewport.GotoBottom()
			return nil
		case key.Matches(msg, lv.keys.PageDown):
			lv.viewport.HalfViewDown()
			return nil
		case key.Matches(msg, lv.keys.PageUp):
			lv.viewport.HalfViewUp()
			return nil
		}
	}

	// Update viewport (handles scrolling with arrow keys and page up/down)
	lv.viewport, cmd = lv.viewport.Update(msg)
	return cmd
}

// View renders the logs view.
func (lv *LogsView) View() string {
	if !lv.ready {
		return "Loading logs..."
	}

	if len(lv.lines) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Render("Waiting for log entries...")
	}

	return lv.viewport.View()
}

// Stop is a no-op for the logs view (kept for interface compatibility).
func (lv *LogsView) Stop() {
	// No cleanup needed
}
