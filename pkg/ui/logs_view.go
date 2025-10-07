package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bjartek/aether/pkg/logs"
)

// LogsView manages the logs display with a scrollable viewport.
type LogsView struct {
	viewport viewport.Model
	lines    []string
	maxLines int
	ready    bool
}

// NewLogsView creates a new logs view.
func NewLogsView() *LogsView {
	return &LogsView{
		lines:    make([]string, 0),
		maxLines: 1000, // Keep last 1000 lines in memory
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
			// Add new line
			lv.lines = append(lv.lines, strings.TrimRight(msg.Line, "\n"))
			
			// Keep only the last maxLines
			if len(lv.lines) > lv.maxLines {
				lv.lines = lv.lines[len(lv.lines)-lv.maxLines:]
			}
			
			// Update viewport content
			if lv.ready {
				lv.viewport.SetContent(strings.Join(lv.lines, "\n"))
				// Auto-scroll to bottom
				lv.viewport.GotoBottom()
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
	}

	// Update viewport (handles scrolling)
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
